package types

import (
	"context"
	"time"

	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types "github.com/filecoin-project/venus/venus-shared/types/gateway"
)

type ChannelInfo struct {
	Ctx        context.Context
	ChannelId  sharedTypes.UUID
	Ip         string
	OutBound   chan *types.RequestEvent
	CreateTime time.Time
}

func NewChannelInfo(ctx context.Context, ip string, sendEvents chan *types.RequestEvent) *ChannelInfo {
	return &ChannelInfo{
		Ctx:        ctx,
		ChannelId:  sharedTypes.NewUUID(),
		OutBound:   sendEvents,
		Ip:         ip,
		CreateTime: time.Now(),
	}
}
