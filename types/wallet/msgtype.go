// Code quoted from github.com/filecoin-project/venus-wallet/core/types.go. DO NOT EDIT.

package wallet

import (
	"math"

	"github.com/filecoin-project/go-state-types/crypto"
)

type SigType = crypto.SigType

const (
	SigTypeUnknown = SigType(math.MaxUint8)

	SigTypeSecp256k1 = SigType(iota)
	SigTypeBLS
)

type MsgType string
type MsgMeta struct {
	Type MsgType
	// Additional data related to what is signed. Should be verifiable with the
	// signed bytes (e.g. CID(Extra).Bytes() == toSign)
	Extra []byte
}

const  (
	MTUnknown       = MsgType("")
	MTVerifyAddress = MsgType("verifyaddress")
)
