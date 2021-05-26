package walletevent

import (
	"github.com/filecoin-project/go-address"
	"github.com/google/uuid"
	"github.com/ipfs-force-community/venus-gateway/types"
)

type walletPayloadRequest struct {
	Account string
	Addr    address.Address
	Method  string
	Payload []byte
	Result  chan *types.ResponseEvent
}

type WalletDetail struct {
	Account        string
	SupportAccount []string
	ConnectStates  []ConnectState
}

type ConnectState struct {
	Addrs        []address.Address
	ChannelId    uuid.UUID
	Ip           string
	RequestCount int
}
