package proofevent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/venus-auth/jwtclient"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	"github.com/filecoin-project/venus/venus-shared/api/gateway/v1"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types2 "github.com/filecoin-project/venus/venus-shared/types/gateway"
	"github.com/ipfs-force-community/venus-gateway/types"
	"github.com/ipfs-force-community/venus-gateway/validator"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("proof_stream")

var _ gateway.IProofClient = (*ProofEventStream)(nil)

type ProofEventStream struct {
	connLk           sync.RWMutex
	minerConnections map[address.Address]*channelStore
	cfg              *types.RequestConfig
	validator        validator.IAuthMinerValidator
	*types.BaseEventStream
}

func NewProofEventStream(ctx context.Context, validator validator.IAuthMinerValidator, cfg *types.RequestConfig) *ProofEventStream {
	proofEventStream := &ProofEventStream{
		connLk:           sync.RWMutex{},
		minerConnections: make(map[address.Address]*channelStore),
		cfg:              cfg,
		validator:        validator,
		BaseEventStream:  types.NewBaseEventStream(ctx, cfg),
	}
	return proofEventStream
}

func (e *ProofEventStream) ListenProofEvent(ctx context.Context, policy *types2.ProofRegisterPolicy) (<-chan *types2.RequestEvent, error) {
	ip, exist := jwtclient.CtxGetTokenLocation(ctx)
	if !exist {
		return nil, fmt.Errorf("ip not exist")
	}
	err := e.validator.Validate(ctx, policy.MinerAddress)
	if err != nil {
		return nil, fmt.Errorf("verify miner:%s failed:%w", policy.MinerAddress.String(), err)
	}

	out := make(chan *types2.RequestEvent, e.cfg.RequestQueueSize)
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
		connectBytes, err := json.Marshal(types2.ConnectedCompleted{
			ChannelId: channel.ChannelId,
		})
		if err != nil {
			close(out)
			log.Errorf("marshal failed %v", err)
			return
		}

		out <- &types2.RequestEvent{
			ID:         sharedTypes.NewUUID(),
			Method:     "InitConnect",
			Payload:    connectBytes,
			CreateTime: time.Now(),
			Result:     nil,
		} // no response
		defer close(out)
		<-ctx.Done()
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
	}()
	return out, nil
}

func (e *ProofEventStream) ResponseProofEvent(ctx context.Context, resp *types2.ResponseEvent) error {
	return e.ResponseEvent(ctx, resp)
}

func (e *ProofEventStream) ComputeProof(ctx context.Context, miner address.Address, sectorInfos []builtin.ExtendedSectorInfo, rand abi.PoStRandomness, height abi.ChainEpoch, nwVersion network.Version) ([]builtin.PoStProof, error) {
	reqBody := types2.ComputeProofRequest{
		SectorInfos: sectorInfos,
		Rand:        rand,
		Height:      height,
		NWVersion:   nwVersion,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	channels, err := e.getChannels(miner)
	if err != nil {
		return nil, err
	}
	var result []builtin.PoStProof
	err = e.SendRequest(ctx, channels, "ComputeProof", payload, &result)
	if err == nil {
		return result, nil
	}
	return nil, err
}

func (e *ProofEventStream) getChannels(mAddr address.Address) ([]*types.ChannelInfo, error) {
	e.connLk.Lock()
	var channelStore *channelStore
	var ok bool
	if channelStore, ok = e.minerConnections[mAddr]; !ok {
		e.connLk.Unlock()
		return nil, fmt.Errorf("no connections for this miner %s", mAddr)
	}
	e.connLk.Unlock()

	channels, err := channelStore.getChannelListByMiners()
	if err != nil {
		return nil, fmt.Errorf("cannot find any connection for miner %s", mAddr)
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

func (e *ProofEventStream) ListMinerConnection(ctx context.Context, addr address.Address) (*types2.MinerState, error) {
	e.connLk.Lock()
	defer e.connLk.Unlock()

	if store, ok := e.minerConnections[addr]; ok {
		return store.getChannelState(), nil
	}
	return nil, fmt.Errorf("miner %s not exit", addr)
}
