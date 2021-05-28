package proofevent

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-actors/actors/runtime/proof"
	"github.com/ipfs-force-community/venus-gateway/types"
)

type IProofEventAPI interface {
	ResponseProofEvent(ctx context.Context, resp *types.ResponseEvent) error
	ListenProofEvent(ctx context.Context, policy *ProofRegisterPolicy) (chan *types.RequestEvent, error)
}

type IProofEvent interface {
	ListConnectedMiners(ctx context.Context) ([]address.Address, error)
	ListMinerConnection(ctx context.Context, addr address.Address) (*MinerState, error)

	ComputeProof(ctx context.Context, miner address.Address, sectorInfos []proof.SectorInfo, rand abi.PoStRandomness) ([]proof.PoStProof, error)
}

var _ IProofEventAPI = (*ProofEventAPI)(nil)

type ProofEventAPI struct {
	proofEvent *ProofEventStream
}

func NewProofEventAPI(proofEvent *ProofEventStream) *ProofEventAPI {
	return &ProofEventAPI{proofEvent: proofEvent}
}

func (proofEventAPI *ProofEventAPI) ResponseProofEvent(ctx context.Context, resp *types.ResponseEvent) error {
	return proofEventAPI.proofEvent.ResponseEvent(ctx, resp)
}

func (proofEventAPI *ProofEventAPI) ListenProofEvent(ctx context.Context, policy *ProofRegisterPolicy) (chan *types.RequestEvent, error) {
	return proofEventAPI.proofEvent.ListenProofEvent(ctx, policy)
}
