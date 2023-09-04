package membership

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	v2API "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"
	types "github.com/filecoin-project/venus/venus-shared/types"
	gtypes "github.com/filecoin-project/venus/venus-shared/types/gateway"
	"github.com/ipfs/go-cid"
)

var _ v2API.IWalletClient = (*Node)(nil)

func (c *Node) ListWalletInfo(ctx context.Context) ([]*gtypes.WalletDetail, error) {
	panic("implement me")
}

func (c *Node) ListWalletInfoByWallet(ctx context.Context, wallet string) (*gtypes.WalletDetail, error) {
	panic("implement me")
}

func (c *Node) WalletHas(ctx context.Context, addr address.Address, accounts []string) (bool, error) {
	panic("implement me")
}

func (c *Node) WalletSign(ctx context.Context, addr address.Address, accounts []string, toSign []byte, meta types.MsgMeta) (*crypto.Signature, error) {
	panic("implement me")
}

var _ v2API.IMarketClient = (*Node)(nil)

func (c *Node) ListMarketConnectionsState(ctx context.Context) ([]gtypes.MarketConnectionState, error) {
	panic("implement me")
}

func (c *Node) SectorsUnsealPiece(ctx context.Context, miner address.Address, pieceCid cid.Cid, sid abi.SectorNumber, offset types.UnpaddedByteIndex, size abi.UnpaddedPieceSize, dest string) (gtypes.UnsealState, error) {
	panic("implement me")
}

var _ v2API.IProofClient = (*Node)(nil)

func (c *Node) ListConnectedMiners(ctx context.Context) ([]address.Address, error) {
	panic("implement me")
}

func (c *Node) ListMinerConnection(ctx context.Context, addr address.Address) (*gtypes.MinerState, error) {
	panic("implement me")
}
func (c *Node) ComputeProof(ctx context.Context, miner address.Address, sectorInfos []builtin.ExtendedSectorInfo, rand abi.PoStRandomness, height abi.ChainEpoch, nwVersion network.Version) ([]builtin.PoStProof, error) {
	panic("implement me")
}
