package marketevent

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ipfs-force-community/venus-gateway/validator"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus-auth/cmd/jwtclient"

	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types2 "github.com/filecoin-project/venus/venus-shared/types/gateway"
	"github.com/ipfs-force-community/venus-gateway/types"
)

var log = logging.Logger("market_stream")

var _ IMarketEvent = (*MarketEventStream)(nil)

type MarketEventStream struct {
	connLk           sync.RWMutex
	minerConnections map[address.Address]*channelStore
	cfg              *types.Config
	validator        validator.IAuthMinerValidator
	*types.BaseEventStream
}

func NewMarketEventStream(ctx context.Context, validator validator.IAuthMinerValidator, cfg *types.Config) *MarketEventStream {
	marketEventStream := &MarketEventStream{
		connLk:           sync.RWMutex{},
		minerConnections: make(map[address.Address]*channelStore),
		cfg:              cfg,
		validator:        validator,
		BaseEventStream:  types.NewBaseEventStream(ctx, cfg),
	}
	return marketEventStream
}

func (e *MarketEventStream) ListenMarketEvent(ctx context.Context, policy *types2.MarketRegisterPolicy) (chan *types2.RequestEvent, error) {
	ip, exist := jwtclient.CtxGetTokenLocation(ctx)
	if !exist {
		return nil, fmt.Errorf("ip not exist")
	}
	err := e.validator.Validate(ctx, policy.Miner)
	if err != nil {
		return nil, xerrors.Errorf("verify miner:%s failed:%w", policy.Miner.String(), err)
	}

	out := make(chan *types2.RequestEvent, e.cfg.RequestQueueSize)
	channel := types.NewChannelInfo(ip, out)
	mAddr := policy.Miner
	e.connLk.Lock()
	var channelStore *channelStore
	var ok bool
	if channelStore, ok = e.minerConnections[mAddr]; !ok {
		channelStore = newChannelStore()
		e.minerConnections[policy.Miner] = channelStore
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

func (e *MarketEventStream) ListMarketConnectionsState(ctx context.Context) ([]types2.MarketConnectionState, error) {
	var result []types2.MarketConnectionState
	for addr, conn := range e.minerConnections {
		result = append(result, types2.MarketConnectionState{
			Addr: addr,
			Conn: *conn.getChannelState(),
		})
	}
	return result, nil
}

func (e *MarketEventStream) IsUnsealed(ctx context.Context, miner address.Address, pieceCid cid.Cid, sector storage.SectorRef, offset sharedTypes.PaddedByteIndex, size abi.PaddedPieceSize) (bool, error) {
	reqBody := types2.IsUnsealRequest{
		PieceCid: pieceCid,
		Sector:   sector,
		Offset:   offset,
		Size:     size,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return false, err
	}

	channels, err := e.getChannels(miner)
	if err != nil {
		return false, err
	}
	var result bool
	err = e.SendRequest(ctx, channels, "IsUnsealed", payload, &result)
	if err == nil {
		return result, nil
	} else {
		return false, err
	}
}

func (e *MarketEventStream) SectorsUnsealPiece(ctx context.Context, miner address.Address, pieceCid cid.Cid, sector storage.SectorRef, offset sharedTypes.PaddedByteIndex, size abi.PaddedPieceSize, dest string) error {
	reqBody := types2.UnsealRequest{
		PieceCid: pieceCid,
		Sector:   sector,
		Offset:   offset,
		Size:     size,
		Dest:     dest,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	channels, err := e.getChannels(miner)
	if err != nil {
		return err
	}

	return e.SendRequest(ctx, channels, "SectorsUnsealPiece", payload, nil)
}

func (e *MarketEventStream) getChannels(mAddr address.Address) ([]*types.ChannelInfo, error) {
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
