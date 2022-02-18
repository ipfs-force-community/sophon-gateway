package cmds

import (
	"context"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/crypto"

	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types "github.com/filecoin-project/venus/venus-shared/types/gateway"
)

type GatewayAPI struct {
	ListWalletInfoByWallet     func(ctx context.Context, wallet string) (*types.WalletDetail, error)
	ListWalletInfo             func(ctx context.Context) ([]*types.WalletDetail, error)
	ListMinerConnection        func(ctx context.Context, addr address.Address) (*types.MinerState, error)
	ListConnectedMiners        func(ctx context.Context) ([]address.Address, error)
	ListMarketConnectionsState func(ctx context.Context) ([]types.MarketConnectionState, error)
	WalletSign                 func(ctx context.Context, account string, addr address.Address, toSign []byte, meta sharedTypes.MsgMeta) (*crypto.Signature, error)
	WalletHas                  func(ctx context.Context, supportAccount string, addr address.Address) (bool, error)
	ComputeProof               func(ctx context.Context, miner address.Address, reqBody *types.ComputeProofRequest) ([]builtin.PoStProof, error)
}

func NewGatewayClient(ctx *cli.Context) (*GatewayAPI, jsonrpc.ClientCloser, error) {
	var gatewayAPI = &GatewayAPI{}
	listen := ctx.String("listen")
	addr, err := DialArgs(listen)
	if err != nil {
		return nil, nil, err
	}
	header := http.Header{}

	const tokenFile = "./token"
	var token []byte

	if token, err = ioutil.ReadFile(tokenFile); err != nil {
		return nil, nil, err
	}

	header.Add("Authorization", "Bearer "+string(token))

	closer, err := jsonrpc.NewMergeClient(ctx.Context, addr,
		"Gateway", []interface{}{gatewayAPI}, header)
	if err != nil {
		return nil, nil, err
	}
	return gatewayAPI, closer, nil
}

func DialArgs(addr string) (string, error) {
	ma, err := multiaddr.NewMultiaddr(addr)
	if err == nil {
		_, addr, err := manet.DialArgs(ma)
		if err != nil {
			return "", err
		}

		return "ws://" + addr + "/rpc/v0", nil
	}

	_, err = url.Parse(addr)
	if err != nil {
		return "", err
	}
	return addr + "/rpc/v0", nil
}
