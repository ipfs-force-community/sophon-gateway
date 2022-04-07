package marketevent

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types "github.com/filecoin-project/venus/venus-shared/types/gateway"
	"github.com/ipfs/go-cid"
)

//IMarketEventAPI used for client connection
type IMarketEventAPI interface {
	ResponseMarketEvent(ctx context.Context, resp *types.ResponseEvent) error
	ListenMarketEvent(ctx context.Context, policy *types.MarketRegisterPolicy) (<-chan *types.RequestEvent, error)
}

// IMarketEvent: need ListConnectedMiners & ListConnectedMiners ?
type IMarketEvent interface {
	//should use  storiface.UnpaddedByteIndex as type for offset
	IsUnsealed(ctx context.Context, miner address.Address, pieceCid cid.Cid, sector storage.SectorRef, offset sharedTypes.PaddedByteIndex, size abi.PaddedPieceSize) (bool, error)
	// SectorsUnsealPiece will Unseal a Sealed sector file for the given sector.
	//should use  storiface.UnpaddedByteIndex as type for offset
	SectorsUnsealPiece(ctx context.Context, miner address.Address, pieceCid cid.Cid, sector storage.SectorRef, offset sharedTypes.PaddedByteIndex, size abi.PaddedPieceSize, dest string) error
}

var _ IMarketEventAPI = (*MarketEventAPI)(nil)

//MarketEventAPI implement market event api
type MarketEventAPI struct {
	marketEvent *MarketEventStream
}

func NewMarketEventAPI(marketEvent *MarketEventStream) *MarketEventAPI {
	return &MarketEventAPI{marketEvent: marketEvent}
}

func (marketEventAPI *MarketEventAPI) ResponseMarketEvent(ctx context.Context, resp *types.ResponseEvent) error {

	return marketEventAPI.marketEvent.ResponseEvent(ctx, resp)
}

func (marketEventAPI *MarketEventAPI) ListenMarketEvent(ctx context.Context, policy *types.MarketRegisterPolicy) (<-chan *types.RequestEvent, error) {
	return marketEventAPI.marketEvent.ListenMarketEvent(ctx, policy)
}
