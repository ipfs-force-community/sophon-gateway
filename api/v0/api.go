package api

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	api "github.com/filecoin-project/venus/venus-shared/api/gateway/v1"
	types "github.com/filecoin-project/venus/venus-shared/types/gateway"
)

type GatewayFullNode interface {
	IProofEvent
	IWalletEvent
	IMarketEvent
}

type IProofEvent interface {
	ListConnectedMiners(ctx context.Context) ([]address.Address, error)                                                                                      //perm:admin
	ListMinerConnection(ctx context.Context, addr address.Address) (*types.MinerState, error)                                                                //perm:admin
	ComputeProof(ctx context.Context, miner address.Address, sectorInfos []builtin.ExtendedSectorInfo, rand abi.PoStRandomness) ([]builtin.PoStProof, error) //perm:admin

	ResponseProofEvent(ctx context.Context, resp *types.ResponseEvent) error                                     //perm:read
	ListenProofEvent(ctx context.Context, policy *types.ProofRegisterPolicy) (<-chan *types.RequestEvent, error) //perm:read
}

type IWalletEvent = api.IWalletEvent

type IMarketEvent = api.IMarketEvent
