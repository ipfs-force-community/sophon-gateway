package marketevent

import (
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"
	types2 "github.com/ipfs-force-community/venus-common-utils/types"
	"sync"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/google/uuid"
	"github.com/ipfs-force-community/venus-gateway/types"
	"golang.org/x/xerrors"
)

type channelStore struct {
	channels map[uuid.UUID]*types.ChannelInfo
	lk       sync.RWMutex
}

func newChannelStore() *channelStore {
	return &channelStore{
		channels: make(map[uuid.UUID]*types.ChannelInfo),
		lk:       sync.RWMutex{},
	}
}

// nolint
func (cs *channelStore) getChannelByMiners() (*types.ChannelInfo, error) {
	cs.lk.RLock()
	defer cs.lk.RUnlock()
	for _, channel := range cs.channels {
		return channel, nil
	}
	return nil, xerrors.Errorf("no any connection")
}

func (cs *channelStore) getChannelListByMiners() ([]*types.ChannelInfo, error) {
	cs.lk.RLock()
	defer cs.lk.RUnlock()
	if len(cs.channels) == 0 {
		return nil, xerrors.Errorf("no any connection")
	}
	var channels []*types.ChannelInfo
	for _, channel := range cs.channels {
		channels = append(channels, channel)
	}
	return channels, nil
}

func (cs *channelStore) addChanel(ch *types.ChannelInfo) error {
	cs.lk.Lock()
	defer cs.lk.Unlock()

	cs.channels[ch.ChannelId] = ch
	return nil
}

func (cs *channelStore) removeChanel(ch *types.ChannelInfo) error {
	cs.lk.Lock()
	defer cs.lk.Unlock()

	delete(cs.channels, ch.ChannelId)
	return nil
}

func (cs *channelStore) getChannelState() *MinerState {
	cs.lk.Lock()
	defer cs.lk.Unlock()
	cstate := &MinerState{}
	for chid, chanStore := range cs.channels {
		cstate.ConnectionCount++
		cstate.Connections = append(cstate.Connections, &ConnectState{
			Channel:      chid,
			RequestCount: len(chanStore.OutBound),
			Ip:           chanStore.Ip,
			CreateTime:   chanStore.CreateTime,
		})
	}
	return cstate
}

func (cs *channelStore) empty() bool {
	cs.lk.Lock()
	defer cs.lk.Unlock()
	return len(cs.channels) == 0
}

type ConnectState struct {
	Channel      uuid.UUID
	Ip           string
	RequestCount int
	CreateTime   time.Time
}

type MinerState struct {
	Connections     []*ConnectState
	ConnectionCount int
}

type MarketRegisterPolicy struct {
	Miner address.Address
}

type IsUnsealRequest struct {
	Sector storage.SectorRef
	Offset types2.PaddedByteIndex
	Size   abi.PaddedPieceSize
}

type IsUnsealResponse struct {
}

type UnsealRequest struct {
	Sector storage.SectorRef
	Offset types2.PaddedByteIndex
	Size   abi.PaddedPieceSize
	Dest   string
}

type UnsealResponse struct {
}
