package walletevent

import (
	"github.com/filecoin-project/go-address"
	"github.com/google/uuid"
	"time"
)

type WalletDetail struct {
	Account         string
	SupportAccounts []string
	ConnectStates   []ConnectState
}

type ConnectState struct {
	Addrs        []address.Address
	ChannelId    uuid.UUID
	Ip           string
	RequestCount int
	CreateTime   time.Time
}

type WalletRegisterPolicy struct {
	SupportAccounts []string
}
