package marketevent

import (
	"context"
	types2 "github.com/ipfs-force-community/venus-common-utils/types"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"
	"github.com/ipfs-force-community/venus-gateway/types"
)

type IMarketEventAPI interface {
	ResponseMarketEvent(ctx context.Context, resp *types.ResponseEvent) error
	ListenMarketEvent(ctx context.Context, policy *MarketRegisterPolicy) (chan *types.RequestEvent, error)
}

type IMarketEvent interface {
	ListConnectedMiners(ctx context.Context) ([]address.Address, error)
	ListMinerConnection(ctx context.Context, addr address.Address) (*MinerState, error)

	//should use  storiface.UnpaddedByteIndex as type for offset
	IsUnsealed(ctx context.Context, miner address.Address, sector storage.SectorRef, offset types2.PaddedByteIndex, size abi.PaddedPieceSize) (bool, error)
	// SectorsUnsealPiece will Unseal a Sealed sector file for the given sector.
	//should use  storiface.UnpaddedByteIndex as type for offset
	SectorsUnsealPiece(ctx context.Context, miner address.Address, sector storage.SectorRef, offset types2.PaddedByteIndex, size abi.PaddedPieceSize, randomness abi.SealRandomness, commd cid.Cid, dest string) error
}

var _ IMarketEventAPI = (*MarketEventAPI)(nil)

type MarketEventAPI struct {
	marketEvent *MarketEventStream
}

func NewMarketEventAPI(marketEvent *MarketEventStream) *MarketEventAPI {
	return &MarketEventAPI{marketEvent: marketEvent}
}

func (marketEventAPI *MarketEventAPI) ResponseMarketEvent(ctx context.Context, resp *types.ResponseEvent) error {

	return marketEventAPI.marketEvent.ResponseEvent(ctx, resp)
}

func (marketEventAPI *MarketEventAPI) ListenMarketEvent(ctx context.Context, policy *MarketRegisterPolicy) (chan *types.RequestEvent, error) {
	return marketEventAPI.marketEvent.ListenMarketEvent(ctx, policy)
}
