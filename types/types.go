package types

import (
	"time"

	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types "github.com/filecoin-project/venus/venus-shared/types/gateway"
)

type ChannelInfo struct {
	ChannelId  sharedTypes.UUID
	Ip         string
	OutBound   chan *types.RequestEvent
	CreateTime time.Time
}

func NewChannelInfo(ip string, sendEvents chan *types.RequestEvent) *ChannelInfo {
	return &ChannelInfo{
		ChannelId:  sharedTypes.NewUUID(),
		OutBound:   sendEvents,
		Ip:         ip,
		CreateTime: time.Now(),
	}
}
