package api

import (
	"context"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/network"

	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	v2API "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	gtypes "github.com/filecoin-project/venus/venus-shared/types/gateway"

	"github.com/ipfs-force-community/sophon-gateway/marketevent"
	"github.com/ipfs-force-community/sophon-gateway/proofevent"
	"github.com/ipfs-force-community/sophon-gateway/version"
	"github.com/ipfs-force-community/sophon-gateway/walletevent"
)

type IGatewayPushAPI interface {
	v2API.IProofClient
	v2API.IWalletClient
}

type IGatewayAPI interface {
	IGatewayPushAPI

	v2API.IProofServiceProvider
	v2API.IWalletServiceProvider
}

var (
	_ v2API.IGateway = (*GatewayAPIImpl)(nil)
	_ IGatewayAPI    = (*GatewayAPIImpl)(nil)
)

type GatewayAPIImpl struct {
	v2API.IProofServiceProvider
	pe *proofevent.ProofEventStream

	v2API.IWalletServiceProvider
	we *walletevent.WalletEventStream

	v2API.IMarketServiceProvider

	me *marketevent.MarketEventStream
}

func NewGatewayAPIImpl(pe *proofevent.ProofEventStream, we *walletevent.WalletEventStream, me *marketevent.MarketEventStream) *GatewayAPIImpl {
	return &GatewayAPIImpl{
		pe: pe,
		we: we,
		me: me,

		IProofServiceProvider:  pe,
		IWalletServiceProvider: we,
		IMarketServiceProvider: me,
	}
}

func (g *GatewayAPIImpl) ComputeProof(ctx context.Context, miner address.Address, sectorInfos []builtin.ExtendedSectorInfo, rand abi.PoStRandomness, height abi.ChainEpoch, nwVersion network.Version) ([]builtin.PoStProof, error) {
	return g.pe.ComputeProof(ctx, miner, sectorInfos, rand, height, nwVersion)
}

func (g *GatewayAPIImpl) ListConnectedMiners(ctx context.Context) ([]address.Address, error) {
	return g.pe.ListConnectedMiners(ctx)
}

func (g *GatewayAPIImpl) ListMinerConnection(ctx context.Context, addr address.Address) (*gtypes.MinerState, error) {
	return g.pe.ListMinerConnection(ctx, addr)
}

func (g *GatewayAPIImpl) WalletHas(ctx context.Context, addr address.Address, accounts []string) (bool, error) {
	return g.we.WalletHas(ctx, addr, accounts)
}

func (g *GatewayAPIImpl) WalletSign(ctx context.Context, addr address.Address, accounts []string, toSign []byte, meta sharedTypes.MsgMeta) (*crypto.Signature, error) {
	return g.we.WalletSign(ctx, addr, accounts, toSign, meta)
}

func (g *GatewayAPIImpl) ListWalletInfo(ctx context.Context) ([]*gtypes.WalletDetail, error) {
	return g.we.ListWalletInfo(ctx)
}

func (g *GatewayAPIImpl) ListWalletInfoByWallet(ctx context.Context, wallet string) (*gtypes.WalletDetail, error) {
	return g.we.ListWalletInfoByWallet(ctx, wallet)
}

func (g *GatewayAPIImpl) SectorsUnsealPiece(ctx context.Context, miner address.Address, pieceCid cid.Cid, sid abi.SectorNumber, offset sharedTypes.UnpaddedByteIndex, size abi.UnpaddedPieceSize, dest string) (gtypes.UnsealState, error) {
	return g.me.SectorsUnsealPiece(ctx, miner, pieceCid, sid, offset, size, dest)
}

func (g *GatewayAPIImpl) ListMarketConnectionsState(ctx context.Context) ([]gtypes.MarketConnectionState, error) {
	return g.me.ListMarketConnectionsState(ctx)
}

func (g *GatewayAPIImpl) Version(context.Context) (sharedTypes.Version, error) {
	return sharedTypes.Version{Version: version.UserVersion}, nil
}
