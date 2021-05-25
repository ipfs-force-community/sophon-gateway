package cmds

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/specs-actors/actors/runtime/proof"
	"github.com/filecoin-project/venus-wallet/core"
	"github.com/ipfs-force-community/venus-gateway/proofevent"
	"github.com/ipfs-force-community/venus-gateway/types"
	"github.com/ipfs-force-community/venus-gateway/walletevent"
	"github.com/urfave/cli/v2"
)

type GatewayAPI struct {
	ListWalletInfoByWallet func(ctx context.Context, wallet string) (*walletevent.WalletDetail, error)
	ListWalletInfo         func(ctx context.Context) ([]*walletevent.WalletDetail, error)
	ListMinerConnection    func(ctx context.Context, addr address.Address) (*proofevent.MinerState, error)
	ListConnectedMiners    func(ctx context.Context) ([]address.Address, error)
	WalletSign             func(ctx context.Context, account string, addr address.Address, toSign []byte, meta core.MsgMeta) (*crypto.Signature, error)
	WalletHas              func(ctx context.Context, supportAccount string, addr address.Address) (bool, error)
	ComputeProof           func(ctx context.Context, miner address.Address, reqBody *types.ComputeProofRequest) ([]proof.PoStProof, error)
}

func NewGatewayClient(ctx *cli.Context) (*GatewayAPI, jsonrpc.ClientCloser, error) {
	var gatewayAPI = &GatewayAPI{}
	closer, err := jsonrpc.NewMergeClient(ctx.Context, "ws://127.0.0.1:45132/rpc/v0", "Filecoin", []interface{}{gatewayAPI}, nil)
	if err != nil {
		return nil, nil, err
	}
	return gatewayAPI, closer, nil
}
