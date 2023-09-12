package cluster

import (
	"context"
	"errors"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	v2api "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"
	types "github.com/filecoin-project/venus/venus-shared/types"
	gtypes "github.com/filecoin-project/venus/venus-shared/types/gateway"
	"github.com/hashicorp/memberlist"
	"github.com/ipfs/go-cid"
)

const (
	// CtxKeyBroadcastPrevent if the key inject to context witch means do not spread the request again to prevent broadcast storm
	CtxKeyBroadcastPrevent string = "stop_spread"
)

// Cluster transfer a cluster of gateway node into a gateway client for IWalletClient, IProofClient, IMarketClient
type Cluster struct {
	// authToken used for authorization of other gateway node
	// we assume that all gateway node in the cluster use the same Auth service,
	// witch means that the authToken is the same for all gateway node in the cluster
	authToken string

	*Node
}

func NewCluster(ctx context.Context, api string, listen string, authToken string) (*Cluster, error) {
	node, err := NewNode(api,
		listen)
	if err != nil {
		return nil, err
	}

	return &Cluster{
		Node:      node,
		authToken: authToken,
	}, nil
}

func (c *Cluster) MemberInfos() ([]v2api.MemberInfo, error) {
	ret := make([]v2api.MemberInfo, 0)
	ret = append(ret, v2api.MemberInfo{
		Name:    c.memberShip.LocalNode().Name,
		Address: c.memberShip.LocalNode().Address(),
		Meta:    extractMeta(c.memberShip.LocalNode().Meta),
	})

	c.ForEachMember(func(n *memberlist.Node) {
		ret = append(ret, v2api.MemberInfo{
			Name:    n.Name,
			Address: n.Address(),
			Meta:    extractMeta(n.Meta),
		})
	})

	return ret, nil
}

var _ v2api.IWalletClient = (*Cluster)(nil)

func (c *Cluster) ListWalletInfo(ctx context.Context) ([]*gtypes.WalletDetail, error) {
	panic("implement me")
}

func (c *Cluster) ListWalletInfoByWallet(ctx context.Context, wallet string) (*gtypes.WalletDetail, error) {
	panic("implement me")
}

func (c *Cluster) WalletHas(ctx context.Context, addr address.Address, accounts []string) (bool, error) {
	ctx = SetPreventBroadcast(ctx)
	ret := false
	c.forEachGatewayClient(ctx, func(client v2api.IGateway) {
		has, err := client.WalletHas(ctx, addr, accounts)
		if err != nil {
			log.Warnf("call WalletHas : %s", err)
		}
		if has {
			ret = true
		}
	})
	return ret, nil
}

func (c *Cluster) WalletSign(ctx context.Context, addr address.Address, accounts []string, toSign []byte, meta types.MsgMeta) (sig *crypto.Signature, retErr error) {
	ctx = SetPreventBroadcast(ctx)

	c.forEachGatewayClient(ctx, func(client v2api.IGateway) {
		if sig != nil {
			has, err := client.WalletHas(ctx, addr, accounts)
			if err != nil {
				log.Warnf("call WalletHas error: %s", retErr)
			}
			if has {
				sig, retErr = client.WalletSign(ctx, addr, accounts, toSign, meta)
				if retErr != nil {
					log.Warnf("call WalletSign fail: %s", retErr)
				}
			}
		}
	})
	return sig, retErr
}

var _ v2api.IMarketClient = (*Cluster)(nil)

func (c *Cluster) ListMarketConnectionsState(ctx context.Context) ([]gtypes.MarketConnectionState, error) {
	panic("implement me")
}

func (c *Cluster) SectorsUnsealPiece(ctx context.Context, miner address.Address, pieceCid cid.Cid, sid abi.SectorNumber, offset types.UnpaddedByteIndex, size abi.UnpaddedPieceSize, dest string) (state gtypes.UnsealState, retErr error) {
	ctx = SetPreventBroadcast(ctx)

	c.forEachGatewayClient(ctx, func(client v2api.IGateway) {
		if state == "" {
			state, retErr = client.SectorsUnsealPiece(ctx, miner, pieceCid, sid, offset, size, dest)
			if errors.Is(retErr, gtypes.ErrNoConnection) {
				state = ""
				return
			}
			if retErr != nil {
				log.Warnf("call SectorsUnsealPiece fail: %s", retErr)
			}
		}
	})

	if state == "" && retErr == nil {
		return gtypes.UnsealStateFailed, fmt.Errorf("no gateway node has the miner %s", miner)
	}
	return
}

var _ v2api.IProofClient = (*Cluster)(nil)

func (c *Cluster) ListConnectedMiners(ctx context.Context) ([]address.Address, error) {
	panic("implement me")
}

func (c *Cluster) ListMinerConnection(ctx context.Context, addr address.Address) (*gtypes.MinerState, error) {
	panic("implement me")
}

func (c *Cluster) ComputeProof(ctx context.Context, miner address.Address, sectorInfos []builtin.ExtendedSectorInfo, rand abi.PoStRandomness, height abi.ChainEpoch, nwVersion network.Version) (proof []builtin.PoStProof, retErr error) {
	ctx = SetPreventBroadcast(ctx)

	c.forEachGatewayClient(ctx, func(client v2api.IGateway) {
		if proof == nil && retErr == nil {
			proof, retErr = client.ComputeProof(ctx, miner, sectorInfos, rand, height, nwVersion)
			if errors.Is(retErr, gtypes.ErrNoConnection) {
				// prevent proof returned is empty but not nil
				proof = nil
				return
			} else if retErr != nil {
				log.Warnf("call ComputeProof fail: %s", retErr)
			}
		}
	})
	if proof == nil && retErr == nil {
		return proof, fmt.Errorf("no gateway node has the miner %s", miner)
	}
	return
}

func (c *Cluster) forEachGatewayClient(ctx context.Context, cb func(client v2api.IGateway)) {
	c.Node.ForEachMember(func(n *memberlist.Node) {
		api := getApiFromMeta(n.Meta)
		client, closer, err := v2api.DialIGatewayRPC(ctx, api, c.authToken, nil)
		if err != nil {
			log.Warnf("connect to gateway(%s) fail: %w", api, err)
		}
		defer closer()
		cb(client)
	})
}

// PreventBroadcast indicate weather we should pass request to other node
func PreventBroadcast(ctx context.Context) bool {
	v := ctx.Value(CtxKeyBroadcastPrevent)
	if v == nil {
		return false
	}
	return v.(bool)
}

func SetPreventBroadcast(ctx context.Context) context.Context {
	return context.WithValue(ctx, CtxKeyBroadcastPrevent, true)
}
