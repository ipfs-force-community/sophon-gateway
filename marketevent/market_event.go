package marketevent

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/ipfs-force-community/sophon-auth/core"

	v2API "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	gtypes "github.com/filecoin-project/venus/venus-shared/types/gateway"

	"github.com/ipfs-force-community/sophon-gateway/metrics"
	"github.com/ipfs-force-community/sophon-gateway/types"
	"github.com/ipfs-force-community/sophon-gateway/validator"
)

var log = logging.Logger("market_stream")

var _ v2API.IMarketClient = (*MarketEventStream)(nil)

type MarketEventStream struct {
	connLk           sync.RWMutex
	minerConnections map[address.Address]*channelStore
	cfg              *types.RequestConfig
	validator        validator.IAuthMinerValidator
	*types.BaseEventStream
}

func NewMarketEventStream(ctx context.Context, validator validator.IAuthMinerValidator, cfg *types.RequestConfig) *MarketEventStream {
	marketEventStream := &MarketEventStream{
		connLk:           sync.RWMutex{},
		minerConnections: make(map[address.Address]*channelStore),
		cfg:              cfg,
		validator:        validator,
		BaseEventStream:  types.NewBaseEventStream(ctx, cfg),
	}
	return marketEventStream
}

func (m *MarketEventStream) ListenMarketEvent(ctx context.Context, policy *gtypes.MarketRegisterPolicy) (<-chan *gtypes.RequestEvent, error) {
	ip, exist := core.CtxGetTokenLocation(ctx)
	if !exist {
		return nil, fmt.Errorf("ip not exist")
	}

	// Chain services serve those miners should be controlled by themselves,so the user and miner cannot be forcibly bound here.
	err := m.validator.Validate(ctx, policy.Miner)
	if err != nil {
		return nil, fmt.Errorf("verify miner:%s failed:%w", policy.Miner.String(), err)
	}

	out := make(chan *gtypes.RequestEvent, m.cfg.RequestQueueSize)
	channel := types.NewChannelInfo(ctx, ip, out)
	mAddr := policy.Miner
	m.connLk.Lock()
	var channelStore *channelStore
	var ok bool
	if channelStore, ok = m.minerConnections[mAddr]; !ok {
		channelStore = newChannelStore()
		m.minerConnections[policy.Miner] = channelStore
	}

	m.connLk.Unlock()
	_ = channelStore.addChanel(channel)
	log.Infof("add new connections %s for miner(market) %s", channel.ChannelId, mAddr)
	go func() {
		connectBytes, err := json.Marshal(gtypes.ConnectedCompleted{
			ChannelId: channel.ChannelId,
		})
		defer close(out)
		if err != nil {
			log.Errorf("marshal failed %v", err)
			return
		}

		ctx, _ = tag.New(ctx, tag.Upsert(metrics.IPKey, ip), tag.Upsert(metrics.MinerAddressKey, mAddr.String()),
			tag.Upsert(metrics.MinerTypeKey, "market"))
		stats.Record(ctx, metrics.MinerRegister.M(1))
		stats.Record(ctx, metrics.MinerSource.M(1))

		out <- &gtypes.RequestEvent{
			ID:         sharedTypes.NewUUID(),
			Method:     "InitConnect",
			Payload:    connectBytes,
			CreateTime: time.Now(),
			Result:     nil,
		} // no response
		<-ctx.Done()
		m.connLk.Lock()
		defer m.connLk.Unlock() // connection read and remove should in one lock
		channelStore := m.minerConnections[mAddr]
		_ = channelStore.removeChanel(channel)
		if channelStore.empty() {
			delete(m.minerConnections, mAddr)
		}
		log.Infof("remove connections %s of miner(market) %s", channel.ChannelId, mAddr)
	}()
	return out, nil
}

func (m *MarketEventStream) ResponseMarketEvent(ctx context.Context, resp *gtypes.ResponseEvent) error {
	return m.ResponseEvent(ctx, resp)
}

func (m *MarketEventStream) ListMarketConnectionsState(ctx context.Context) ([]gtypes.MarketConnectionState, error) {
	var result []gtypes.MarketConnectionState
	for addr, conn := range m.minerConnections {
		result = append(result, gtypes.MarketConnectionState{
			Addr: addr,
			Conn: *conn.getChannelState(),
		})
	}
	return result, nil
}

func (m *MarketEventStream) SectorsUnsealPiece(ctx context.Context, miner address.Address, pieceCid cid.Cid, sid abi.SectorNumber, offset sharedTypes.UnpaddedByteIndex, size abi.UnpaddedPieceSize, dest string) (gtypes.UnsealState, error) {
	reqBody := gtypes.UnsealRequest{
		PieceCid: pieceCid,
		Miner:    miner,
		Sid:      sid,
		Offset:   offset,
		Size:     size,
		Dest:     dest,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return gtypes.UnsealStateFailed, err
	}

	channels, err := m.getChannels(miner)
	if err != nil {
		return gtypes.UnsealStateFailed, err
	}

	start := time.Now()
	var state gtypes.UnsealState
	err = m.SendRequest(ctx, channels, "SectorsUnsealPiece", payload, &state)
	_ = stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(metrics.MinerAddressKey, miner.String())},
		metrics.SectorsUnsealPiece.M(metrics.SinceInMilliseconds(start)))

	return state, err
}

func (m *MarketEventStream) getChannels(mAddr address.Address) ([]*types.ChannelInfo, error) {
	m.connLk.Lock()
	var channelStore *channelStore
	var ok bool
	if channelStore, ok = m.minerConnections[mAddr]; !ok {
		m.connLk.Unlock()
		return nil, fmt.Errorf("no connections for this miner %s", mAddr)
	}
	m.connLk.Unlock()

	channels, err := channelStore.getChannelListByMiners()
	if err != nil {
		return nil, fmt.Errorf("cannot find any connection for miner %s", mAddr)
	}
	return channels, nil
}
