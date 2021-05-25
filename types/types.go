package types

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	proof2 "github.com/filecoin-project/specs-actors/actors/runtime/proof"
	"github.com/filecoin-project/venus-wallet/core"
	"github.com/google/uuid"
	"golang.org/x/xerrors"
)

type Account int

var AccountKey Account

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
	Id      uuid.UUID
	Method  string
	Payload []byte
	Result  chan *ResponseEvent `json:"-"`
}

type ResponseEvent struct {
	Id      uuid.UUID
	Payload []byte
	Error   string
}

type ChannelInfo struct {
	ChannelId uuid.UUID
	OutBound  chan *RequestEvent
}

func NewChannelInfo(sendEvents chan *RequestEvent) *ChannelInfo {
	return &ChannelInfo{
		ChannelId: uuid.New(),
		OutBound:  sendEvents,
	}
}

//request

type ComputeProofRequest struct {
	SectorInfos []proof2.SectorInfo
	Rand        abi.PoStRandomness
}

type WalletConnectedRequest struct {
	ChannelId uuid.UUID
}

type WalletSignRequest struct {
	Signer address.Address
	ToSign []byte
	Meta   core.MsgMeta
}
