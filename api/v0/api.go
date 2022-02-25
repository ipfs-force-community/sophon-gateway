package api

import (
	api "github.com/filecoin-project/venus/venus-shared/api/gateway/v0"
)

type GatewayFullNode interface {
	IProofEvent
	IWalletEvent
	IMarketEvent
}

type IProofEvent = api.IProofEvent

type IWalletEvent = api.IWalletEvent

type IMarketEvent = api.IMarketEvent
