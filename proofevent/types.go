package proofevent

import (
	"github.com/filecoin-project/go-address"
	"github.com/google/uuid"
	"github.com/ipfs-force-community/venus-gateway/types"
	"golang.org/x/xerrors"
	"sync"
)

type minerPayloadRequest struct {
	Miner   address.Address
	Method  string
	Payload []byte
	Result  chan *types.ResponseEvent
}

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

func (cs *channelStore) selectChannel() (*types.ChannelInfo, error) {
	cs.lk.RLock()
	defer cs.lk.RUnlock()
	for _, channel := range cs.channels {
		return channel, nil
	}
	return nil, xerrors.Errorf("no any connection")
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
	RequestCount int
}

type MinerState struct {
	Connections     []*ConnectState
	ConnectionCount int
}
