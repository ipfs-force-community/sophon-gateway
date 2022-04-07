package main

import (
	"context"
	"fmt"
	"net/http"

	"github.com/filecoin-project/go-state-types/network"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"

	proof5 "github.com/filecoin-project/specs-actors/v5/actors/runtime/proof"

	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
)

type ProofEventClient struct {
	ComputeProof func(ctx context.Context, miner address.Address, sectorInfos []proof5.SectorInfo, rand abi.PoStRandomness, height abi.ChainEpoch, nwVersion network.Version) ([]proof5.PoStProof, error)
}
type WalletEventClient struct {
	WalletHas  func(ctx context.Context, supportAccount string, addr address.Address) (bool, error)
	WalletSign func(ctx context.Context, account string, addr address.Address, toSign []byte, meta sharedTypes.MsgMeta) (*crypto.Signature, error)
}

var url = "ws://127.0.0.1:45132/rpc/v1"
var token = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiR2F0ZVdheUxvY2FsVG9rZW4iLCJwZXJtIjoiYWRtaW4iLCJleHQiOiIifQ.jZOlCBnxZtwc9PsjY7OMnooK6C3PFExvZesWsFrVyCs"

const nameSpace = "Gateway"

var header = http.Header{}

func main() {
	header.Add("Authorization", "Bearer "+token)
	fullnode := struct {
		ProofAPI  ProofEventClient
		WalletAPI WalletEventClient
	}{}

	ctx := context.TODO()
	closer, err := jsonrpc.NewMergeClient(ctx, url, nameSpace,
		[]interface{}{&fullnode.WalletAPI, &fullnode.ProofAPI},
		header)
	if err != nil {
		fmt.Printf("new client failed:%s\n", err.Error())
		return
	}
	defer closer()

	SendComputeProof(ctx, &fullnode.ProofAPI)
	WalletHas(ctx, &fullnode.WalletAPI)
	WalletSign(ctx, &fullnode.WalletAPI)
}

func SendComputeProof(ctx context.Context, cli *ProofEventClient) {
	actorAddr, _ := address.NewIDAddress(7)
	result, err := cli.ComputeProof(ctx, actorAddr, []proof5.SectorInfo{{
		SealProof:    1,
		SectorNumber: 0,
		SealedCID:    cid.Undef,
	}}, []byte{1, 2, 3}, 1, 1)
	if err != nil {
		fmt.Printf("computProof failed:%s\n", err.Error())
		return
	}

	fmt.Println(result)
}

func WalletHas(ctx context.Context, cli *WalletEventClient) {
	actorAddr, _ := address.NewIDAddress(1)
	result, err := cli.WalletHas(ctx, "stest", actorAddr)
	if err != nil {
		fmt.Printf("call wallet has failed:%s\n", err.Error())
		return
	}

	fmt.Println(result)

	result, err = cli.WalletHas(ctx, "wtest2", actorAddr)
	if err != nil {
		fmt.Printf("call wallet has failed:%s\n", err.Error())
		return
	}

	fmt.Println(result)

	actorAddr2, _ := address.NewIDAddress(8)
	result, err = cli.WalletHas(ctx, "wtest2", actorAddr2)
	if err != nil {
		fmt.Printf("call wallet has failed:%s\n", err.Error())
		return
	}

	fmt.Println(result)
}

func WalletSign(ctx context.Context, cli *WalletEventClient) {
	actorAddr, _ := address.NewIDAddress(7)
	result, err := cli.WalletSign(ctx, "wtest",
		actorAddr, []byte{1, 2},
		sharedTypes.MsgMeta{
			Type:  "MTUnknown",
			Extra: nil,
		})
	if err != nil {
		fmt.Printf("call wallet sign failed:%s\n", err.Error())
		return
	}

	fmt.Println(result)
}
