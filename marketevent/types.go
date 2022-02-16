package marketevent

import (
	"sync"

	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types3 "github.com/filecoin-project/venus/venus-shared/types/gateway"
	"github.com/ipfs-force-community/venus-gateway/types"
	"golang.org/x/xerrors"
)

type channelStore struct {
	channels map[sharedTypes.UUID]*types.ChannelInfo
	lk       sync.RWMutex
}

func newChannelStore() *channelStore {
	return &channelStore{
		channels: make(map[sharedTypes.UUID]*types.ChannelInfo),
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

func (cs *channelStore) getChannelState() *types3.ConnectionStates {
	cs.lk.Lock()
	defer cs.lk.Unlock()
	cstate := &types3.ConnectionStates{}
	for chid, chanStore := range cs.channels {
		cstate.ConnectionCount++
		cstate.Connections = append(cstate.Connections, &types3.ConnectState{
			ChannelID:    chid,
			RequestCount: len(chanStore.OutBound),
			IP:           chanStore.Ip,
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
