package api

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	proof5 "github.com/filecoin-project/specs-actors/v5/actors/runtime/proof"
	"github.com/ipfs-force-community/venus-gateway/api"
)

type WrapperV1Full struct {
	api.GatewayFullNode
}

func (w WrapperV1Full) ComputeProof(ctx context.Context, miner address.Address, sectorInfos []proof5.SectorInfo, rand abi.PoStRandomness) ([]proof5.PoStProof, error) {
	return w.GatewayFullNode.ComputeProof(ctx, miner, sectorInfos, rand, 0, 0)
}
