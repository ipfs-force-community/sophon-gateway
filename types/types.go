package types

import (
	"time"

	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types "github.com/filecoin-project/venus/venus-shared/types/gateway"
	"golang.org/x/xerrors"
)

const AccountKey = "account"

type IP int

var IPKey IP

const (
	WalletService = "wallet_service"
	ProofService  = "proof_service"
)

// nolint
func checkService(serviceType string) error {
	switch serviceType {
	case WalletService:
		fallthrough
	case ProofService:
		return nil
	default:
		return xerrors.Errorf("unsupport service type %s", serviceType)
	}
}

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
