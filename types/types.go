package types

import (
	"time"

	"github.com/google/uuid"
	"golang.org/x/xerrors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	proof5 "github.com/filecoin-project/specs-actors/v5/actors/runtime/proof"

	"github.com/ipfs-force-community/venus-gateway/types/wallet"
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

type RequestEvent struct {
	Id         uuid.UUID
	Method     string
	Payload    []byte
	CreateTime time.Time           `json:"-"`
	Result     chan *ResponseEvent `json:"-"`
}

type ResponseEvent struct {
	Id      uuid.UUID
	Payload []byte
	Error   string
}

type ChannelInfo struct {
	ChannelId  uuid.UUID
	Ip         string
	OutBound   chan *RequestEvent
	CreateTime time.Time
}

func NewChannelInfo(ip string, sendEvents chan *RequestEvent) *ChannelInfo {
	return &ChannelInfo{
		ChannelId:  uuid.New(),
		OutBound:   sendEvents,
		Ip:         ip,
		CreateTime: time.Now(),
	}
}

//request

type ComputeProofRequest struct {
	SectorInfos []proof5.SectorInfo
	Rand        abi.PoStRandomness
}

type ConnectedCompleted struct {
	ChannelId uuid.UUID
}

type WalletSignRequest struct {
	Signer address.Address
	ToSign []byte
	Meta   wallet.MsgMeta
}
