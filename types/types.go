package types

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	proof2 "github.com/filecoin-project/specs-actors/actors/runtime/proof"
	"github.com/filecoin-project/venus-wallet/core"
	"github.com/google/uuid"
	"golang.org/x/xerrors"
	"time"
)

type Account int

var AccountKey Account

type IP int

var IPKey IP

const (
	WalletService = "wallet_service"
	ProofService  = "proof_service"
)

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
	ChannelId uuid.UUID
	Ip        string
	OutBound  chan *RequestEvent
}

func NewChannelInfo(ip string, sendEvents chan *RequestEvent) *ChannelInfo {
	return &ChannelInfo{
		ChannelId: uuid.New(),
		OutBound:  sendEvents,
		Ip:        ip,
	}
}

//request

type ComputeProofRequest struct {
	SectorInfos []proof2.SectorInfo
	Rand        abi.PoStRandomness
}

type ConnectedCompleted struct {
	ChannelId uuid.UUID
}

type WalletSignRequest struct {
	Signer address.Address
	ToSign []byte
	Meta   core.MsgMeta
}
