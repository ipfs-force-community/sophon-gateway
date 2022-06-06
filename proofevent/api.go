package proofevent

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"

	"github.com/filecoin-project/venus/venus-shared/actors/builtin"

	types "github.com/filecoin-project/venus/venus-shared/types/gateway"
)

type IProofEventAPI interface {
	ResponseProofEvent(ctx context.Context, resp *types.ResponseEvent) error
	ListenProofEvent(ctx context.Context, policy *types.ProofRegisterPolicy) (<-chan *types.RequestEvent, error)
}

type IProofEvent interface {
	ListConnectedMiners(ctx context.Context) ([]address.Address, error)
	ListMinerConnection(ctx context.Context, addr address.Address) (*types.MinerState, error)

	ComputeProof(ctx context.Context, miner address.Address, sectorInfos []builtin.ExtendedSectorInfo, rand abi.PoStRandomness, height abi.ChainEpoch, nwVersion network.Version) ([]builtin.PoStProof, error)
}
