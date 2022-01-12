package main

import (
	"context"
	"github.com/filecoin-project/go-state-types/network"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/specs-storage/storage"

	proof5 "github.com/filecoin-project/specs-actors/v5/actors/runtime/proof"

	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types2 "github.com/ipfs-force-community/venus-common-utils/types"

	"github.com/ipfs-force-community/venus-gateway/marketevent"
	"github.com/ipfs-force-community/venus-gateway/proofevent"
	"github.com/ipfs-force-community/venus-gateway/walletevent"
)

type IGatewayPushAPI interface {
	proofevent.IProofEvent
	walletevent.IWalletEvent
}

type IGatewayAPI interface {
	proofevent.IProofEventAPI
	walletevent.IWalletEventAPI
	IGatewayPushAPI
}

var _ IGatewayAPI = (*GatewayAPI)(nil)

type GatewayAPI struct {
	proofevent.IProofEventAPI
	pe *proofevent.ProofEventStream

	walletevent.IWalletEventAPI
	we *walletevent.WalletEventStream

	marketevent.IMarketEventAPI
	me *marketevent.MarketEventStream
}

func NewGatewayAPI(pe *proofevent.ProofEventStream, we *walletevent.WalletEventStream, me *marketevent.MarketEventStream) *GatewayAPI {
	return &GatewayAPI{
		IProofEventAPI:  proofevent.NewProofEventAPI(pe),
		IWalletEventAPI: walletevent.NewWalletEventAPI(we),
		IMarketEventAPI: marketevent.NewMarketEventAPI(me),
		pe:              pe,
		we:              we,
		me:              me,
	}
}

func (g *GatewayAPI) ComputeProof(ctx context.Context, miner address.Address, sectorInfos []proof5.SectorInfo, rand abi.PoStRandomness, height abi.ChainEpoch, nwVersion network.Version) ([]proof5.PoStProof, error) {
	return g.pe.ComputeProof(ctx, miner, sectorInfos, rand, height, nwVersion)
}

func (g *GatewayAPI) ListConnectedMiners(ctx context.Context) ([]address.Address, error) {
	return g.pe.ListConnectedMiners(ctx)
}

func (g *GatewayAPI) ListMinerConnection(ctx context.Context, addr address.Address) (*proofevent.MinerState, error) {
	return g.pe.ListMinerConnection(ctx, addr)
}

func (g *GatewayAPI) WalletHas(ctx context.Context, supportAccount string, addr address.Address) (bool, error) {
	return g.we.WalletHas(ctx, supportAccount, addr)
}

func (g *GatewayAPI) WalletSign(ctx context.Context, account string, addr address.Address, toSign []byte, meta sharedTypes.MsgMeta) (*crypto.Signature, error) {
	return g.we.WalletSign(ctx, account, addr, toSign, meta)
}

func (g *GatewayAPI) ListWalletInfo(ctx context.Context) ([]*walletevent.WalletDetail, error) {
	return g.we.ListWalletInfo(ctx)
}

func (g *GatewayAPI) ListWalletInfoByWallet(ctx context.Context, wallet string) (*walletevent.WalletDetail, error) {
	return g.we.ListWalletInfoByWallet(ctx, wallet)
}

func (g *GatewayAPI) IsUnsealed(ctx context.Context, miner address.Address, pieceCid cid.Cid, sector storage.SectorRef, offset types2.PaddedByteIndex, size abi.PaddedPieceSize) (bool, error) {
	return g.me.IsUnsealed(ctx, miner, pieceCid, sector, offset, size)
}

func (g *GatewayAPI) SectorsUnsealPiece(ctx context.Context, miner address.Address, pieceCid cid.Cid, sector storage.SectorRef, offset types2.PaddedByteIndex, size abi.PaddedPieceSize, dest string) error {
	return g.me.SectorsUnsealPiece(ctx, miner, pieceCid, sector, offset, size, dest)
}

func (g *GatewayAPI) ListMarketConnectionsState(ctx context.Context) ([]marketevent.MarketConnectionState, error) {
	return g.me.ListMarketConnectionsState(ctx)
}
