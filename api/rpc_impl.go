package api

import (
	"context"
	"errors"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/network"

	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	v2API "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	gtypes "github.com/filecoin-project/venus/venus-shared/types/gateway"

	"github.com/ipfs-force-community/sophon-gateway/cluster"
	"github.com/ipfs-force-community/sophon-gateway/marketevent"
	"github.com/ipfs-force-community/sophon-gateway/proofevent"
	"github.com/ipfs-force-community/sophon-gateway/proxy"
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
	proxy   proxy.IProxy
	cluster *cluster.Cluster

	v2API.IProofServiceProvider
	pe *proofevent.ProofEventStream

	v2API.IWalletServiceProvider
	we *walletevent.WalletEventStream

	v2API.IMarketServiceProvider

	me *marketevent.MarketEventStream
}

func NewGatewayAPIImpl(pe *proofevent.ProofEventStream, we *walletevent.WalletEventStream, me *marketevent.MarketEventStream, p proxy.IProxy, cluster *cluster.Cluster) *GatewayAPIImpl {
	return &GatewayAPIImpl{
		pe:      pe,
		we:      we,
		me:      me,
		proxy:   p,
		cluster: cluster,

		IProofServiceProvider:  pe,
		IWalletServiceProvider: we,
		IMarketServiceProvider: me,
	}
}

func (g *GatewayAPIImpl) ComputeProof(ctx context.Context, miner address.Address, sectorInfos []builtin.ExtendedSectorInfo, rand abi.PoStRandomness, height abi.ChainEpoch, nwVersion network.Version) ([]builtin.PoStProof, error) {
	ret, err := g.pe.ComputeProof(ctx, miner, sectorInfos, rand, height, nwVersion)
	if !cluster.PreventBroadcast(ctx) && errors.Is(err, gtypes.ErrNoConnection) {
		return g.cluster.ComputeProof(ctx, miner, sectorInfos, rand, height, nwVersion)
	}
	return ret, err
}

func (g *GatewayAPIImpl) ListConnectedMiners(ctx context.Context) ([]address.Address, error) {
	return g.pe.ListConnectedMiners(ctx)
}

func (g *GatewayAPIImpl) ListMinerConnection(ctx context.Context, addr address.Address) (*gtypes.MinerState, error) {
	return g.pe.ListMinerConnection(ctx, addr)
}

func (g *GatewayAPIImpl) WalletHas(ctx context.Context, addr address.Address, accounts []string) (bool, error) {
	localExist, err := g.we.WalletHas(ctx, addr, accounts)
	if !cluster.PreventBroadcast(ctx) {
		if errors.Is(err, gtypes.ErrNoConnection) || !localExist {
			return g.cluster.WalletHas(ctx, addr, accounts)
		}
	}
	return localExist, err
}

func (g *GatewayAPIImpl) WalletSign(ctx context.Context, addr address.Address, accounts []string, toSign []byte, meta sharedTypes.MsgMeta) (*crypto.Signature, error) {
	ret, err := g.we.WalletSign(ctx, addr, accounts, toSign, meta)
	if !cluster.PreventBroadcast(ctx) && errors.Is(err, gtypes.ErrNoConnection) {
		return g.cluster.WalletSign(ctx, addr, accounts, toSign, meta)
	}
	return ret, err
}

func (g *GatewayAPIImpl) ListWalletInfo(ctx context.Context) ([]*gtypes.WalletDetail, error) {
	return g.we.ListWalletInfo(ctx)
}

func (g *GatewayAPIImpl) ListWalletInfoByWallet(ctx context.Context, wallet string) (*gtypes.WalletDetail, error) {
	return g.we.ListWalletInfoByWallet(ctx, wallet)
}

func (g *GatewayAPIImpl) SectorsUnsealPiece(ctx context.Context, miner address.Address, pieceCid cid.Cid, sid abi.SectorNumber, offset sharedTypes.UnpaddedByteIndex, size abi.UnpaddedPieceSize, dest string) (gtypes.UnsealState, error) {
	ret, err := g.me.SectorsUnsealPiece(ctx, miner, pieceCid, sid, offset, size, dest)
	if !cluster.PreventBroadcast(ctx) && errors.Is(err, gtypes.ErrNoConnection) {
		return g.cluster.SectorsUnsealPiece(ctx, miner, pieceCid, sid, offset, size, dest)
	}
	return ret, err
}

func (g *GatewayAPIImpl) ListMarketConnectionsState(ctx context.Context) ([]gtypes.MarketConnectionState, error) {
	return g.me.ListMarketConnectionsState(ctx)
}

func (g *GatewayAPIImpl) Version(context.Context) (sharedTypes.Version, error) {
	return sharedTypes.Version{Version: version.UserVersion}, nil
}

func (g *GatewayAPIImpl) RegisterReverse(ctx context.Context, hostKey gtypes.HostKey, address string) error {
	return g.proxy.RegisterReverseByAddr(hostKey, address)
}

func (g *GatewayAPIImpl) Join(ctx context.Context, address string) error {
	return g.cluster.Join(address)
}

func (g *GatewayAPIImpl) MemberInfos(ctx context.Context) ([]v2API.MemberInfo, error) {
	return g.cluster.MemberInfos()
}
