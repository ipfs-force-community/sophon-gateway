package proofevent

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/actors/runtime/proof"
	"github.com/google/uuid"
	"github.com/ipfs-force-community/venus-gateway/types"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"
	"sync"
	"time"
)

var log = logging.Logger("proof_stream")

var _ IProofEvent = (*ProofEventStream)(nil)

type ProofEventStream struct {
	connLk           sync.RWMutex
	minerConnections map[address.Address]*channelStore

	reqLk     sync.RWMutex
	idRequest map[uuid.UUID]*types.RequestEvent

	cfg *types.Config
}

func NewProofEventStream(cfg *types.Config) *ProofEventStream {
	return &ProofEventStream{
		connLk:           sync.RWMutex{},
		minerConnections: make(map[address.Address]*channelStore),
		reqLk:            sync.RWMutex{},
		idRequest:        make(map[uuid.UUID]*types.RequestEvent),
		cfg:              cfg,
	}
}

func (e *ProofEventStream) sendRequest(ctx context.Context, req *minerPayloadRequest) error {
	e.connLk.Lock()
	var channelStore *channelStore
	var ok bool
	if channelStore, ok = e.minerConnections[req.Miner]; !ok {
		e.connLk.Unlock()
		return xerrors.Errorf("no connections for this miner %s", req.Miner)
	}
	e.connLk.Unlock()

	selChannel, err := channelStore.selectChannel()
	if err != nil {
		return xerrors.Errorf("cannot find any connection for miner %s", req.Miner)
	}

	id := uuid.New()
	request := &types.RequestEvent{
		Id:         id,
		Method:     req.Method,
		Payload:    req.Payload,
		CreateTime: time.Now(),
		Result:     req.Result,
	}
	e.reqLk.Lock()
	e.idRequest[id] = request
	e.reqLk.Unlock()

	//timeout here
	select {
	case selChannel.OutBound <- request:
		log.Infof("proof send request %s to miner %s", id, req.Miner)
	case <-ctx.Done():
		return xerrors.Errorf("send request cancel by context")
	case <-time.After(e.cfg.RequestTimeout):
		e.reqLk.Lock()
		delete(e.idRequest, id)
		e.reqLk.Unlock()
		return xerrors.Errorf("request %s too long not response", id)
	}

	return nil
}

func (e *ProofEventStream) ListenProofEvent(ctx context.Context, mAddr address.Address) (chan *types.RequestEvent, error) {
	out := make(chan *types.RequestEvent, e.cfg.RequestQueueSize)
	//todo validate mAddr is really belong of this miner
	channel := types.NewChannelInfo(out)

	e.connLk.Lock()
	var channelStore *channelStore
	var ok bool
	if channelStore, ok = e.minerConnections[mAddr]; !ok {
		channelStore = newChannelStore()
		e.minerConnections[mAddr] = channelStore
	}

	e.connLk.Unlock()
	channelStore.addChanel(channel)
	log.Infof("add new connections %s for miner %s", channel.ChannelId, mAddr)
	go func() {
		connectBytes, err := json.Marshal(types.ConnectedCompleted{
			ChannelId: channel.ChannelId,
		})
		if err != nil {
			close(out)
			log.Errorf("marshal failed %v", err)
			return
		}

		out <- &types.RequestEvent{
			Id:         uuid.New(),
			Method:     "InitConnect",
			Payload:    connectBytes,
			CreateTime: time.Now(),
			Result:     nil,
		} //not response
		for {
			select {
			case <-ctx.Done():
				e.connLk.Lock()
				channelStore := e.minerConnections[mAddr]
				e.connLk.Unlock()
				channelStore.removeChanel(channel)
				if channelStore.empty() {
					e.connLk.Lock()
					delete(e.minerConnections, mAddr)
					e.connLk.Unlock()
				}
				log.Info("remove connections %s of miner ", channel.ChannelId, mAddr)
				return
			}
		}
	}()
	return out, nil
}

func (e *ProofEventStream) ResponseEvent(ctx context.Context, resp *types.ResponseEvent) error {
	e.reqLk.Lock()
	event, ok := e.idRequest[resp.Id]
	if ok {
		delete(e.idRequest, resp.Id)
	} else {
		log.Errorf("request id %s not exit %v", resp.Id.String(), resp)
	}
	e.reqLk.Unlock()
	if ok {
		event.Result <- resp
	}
	return nil
}

func (e *ProofEventStream) ComputeProof(ctx context.Context, miner address.Address, sectorInfos []proof.SectorInfo, rand abi.PoStRandomness) ([]proof.PoStProof, error) {
	reqBody := types.ComputeProofRequest{
		SectorInfos: sectorInfos,
		Rand:        rand,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	resultCh := make(chan *types.ResponseEvent)
	req := &minerPayloadRequest{
		Miner:   miner,
		Method:  "ComputeProof",
		Payload: payload,
		Result:  resultCh,
	}

	err = e.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	respEvent := <-resultCh
	if len(respEvent.Error) > 0 {
		return nil, errors.New(respEvent.Error)
	}
	var result []proof.PoStProof
	err = json.Unmarshal(respEvent.Payload, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (e *ProofEventStream) ListConnectedMiners(ctx context.Context) ([]address.Address, error) {
	e.connLk.Lock()
	defer e.connLk.Unlock()
	var miners []address.Address
	for miner, _ := range e.minerConnections {
		miners = append(miners, miner)
	}
	return miners, nil
}

func (e *ProofEventStream) ListMinerConnection(ctx context.Context, addr address.Address) (*MinerState, error) {
	e.connLk.Lock()
	defer e.connLk.Unlock()

	if store, ok := e.minerConnections[addr]; ok {
		return store.getChannelState(), nil
	} else {
		return nil, xerrors.Errorf("miner %s not exit", addr)
	}
}
