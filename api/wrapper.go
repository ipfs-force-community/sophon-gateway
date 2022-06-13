package api

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	sharedGatewayApiV1 "github.com/filecoin-project/venus/venus-shared/api/gateway/v1"
)

type WrapperV1Full struct {
	sharedGatewayApiV1.IGateway
}

func (w WrapperV1Full) ComputeProof(ctx context.Context, miner address.Address, sectorInfos []builtin.ExtendedSectorInfo, rand abi.PoStRandomness) ([]builtin.PoStProof, error) {
	return w.GatewayFullNode.ComputeProof(ctx, miner, sectorInfos, rand, 0, 0)
}
