package proofevent

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/google/uuid"

	proof5 "github.com/filecoin-project/specs-actors/v5/actors/runtime/proof"

	"github.com/filecoin-project/venus-auth/auth"

	"github.com/ipfs-force-community/venus-gateway/types"
)

var log = logging.Logger("proof_stream")

var _ IProofEvent = (*ProofEventStream)(nil)

type ProofEventStream struct {
	connLk           sync.RWMutex
	minerConnections map[address.Address]*channelStore
	cfg              *types.Config
	authClient       types.IAuthClient
	*types.BaseEventStream
}

func NewProofEventStream(ctx context.Context, authClient types.IAuthClient, cfg *types.Config) *ProofEventStream {
	proofEventStream := &ProofEventStream{
		connLk:           sync.RWMutex{},
		minerConnections: make(map[address.Address]*channelStore),
		cfg:              cfg,
		authClient:       authClient,
		BaseEventStream:  types.NewBaseEventStream(ctx, cfg),
	}
	return proofEventStream
}

func (e *ProofEventStream) ListenProofEvent(ctx context.Context, policy *ProofRegisterPolicy) (chan *types.RequestEvent, error) {
	ip := ctx.Value(types.IPKey).(string)
	has, err := e.authClient.HasMiner(&auth.HasMinerRequest{Miner: policy.MinerAddress.String()})
	if err != nil || !has {
		return nil, xerrors.Errorf("address %s not exit", policy.MinerAddress)
	}

	out := make(chan *types.RequestEvent, e.cfg.RequestQueueSize)
	channel := types.NewChannelInfo(ip, out)
	mAddr := policy.MinerAddress
	e.connLk.Lock()
	var channelStore *channelStore
	var ok bool
	if channelStore, ok = e.minerConnections[mAddr]; !ok {
		channelStore = newChannelStore()
		e.minerConnections[policy.MinerAddress] = channelStore
	}

	e.connLk.Unlock()
	_ = channelStore.addChanel(channel)
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
				_ = channelStore.removeChanel(channel)
				if channelStore.empty() {
					e.connLk.Lock()
					delete(e.minerConnections, mAddr)
					e.connLk.Unlock()
				}
				log.Infof("remove connections %s of miner %s", channel.ChannelId, mAddr)
				return
			}
		}
	}()
	return out, nil
}

func (e *ProofEventStream) ComputeProof(ctx context.Context, miner address.Address, sectorInfos []proof5.SectorInfo, rand abi.PoStRandomness) ([]proof5.PoStProof, error) {
	reqBody := types.ComputeProofRequest{
		SectorInfos: sectorInfos,
		Rand:        rand,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	channels, err := e.getChannels(miner)
	if err != nil {
		return nil, err
	}
	var result []proof5.PoStProof
	err = e.SendRequest(ctx, channels, "ComputeProof", payload, &result)
	if err == nil {
		return result, nil
	} else {
		return nil, err
	}
}

func (e *ProofEventStream) getChannels(mAddr address.Address) ([]*types.ChannelInfo, error) {
	e.connLk.Lock()
	var channelStore *channelStore
	var ok bool
	if channelStore, ok = e.minerConnections[mAddr]; !ok {
		e.connLk.Unlock()
		return nil, xerrors.Errorf("no connections for this miner %s", mAddr)
	}
	e.connLk.Unlock()

	channels, err := channelStore.getChannelListByMiners()
	if err != nil {
		return nil, xerrors.Errorf("cannot find any connection for miner %s", mAddr)
	}
	return channels, nil
}

func (e *ProofEventStream) ListConnectedMiners(ctx context.Context) ([]address.Address, error) {
	e.connLk.Lock()
	defer e.connLk.Unlock()
	var miners []address.Address
	for miner := range e.minerConnections {
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
