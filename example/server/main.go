package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"

	proof5 "github.com/filecoin-project/specs-actors/v5/actors/runtime/proof"

	"github.com/ipfs-force-community/venus-gateway/types/wallet"
)

type ProofEventClient struct {
	ComputeProof func(ctx context.Context, miner address.Address, sectorInfos []proof5.SectorInfo, rand abi.PoStRandomness) ([]proof5.PoStProof, error)
}

func main() {
	SendComputeProof()
	WalletHas()
	WalletSign()
}

func SendComputeProof() {
	ctx := context.Background()
	headers := http.Header{}
	headers.Add("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoic3Rlc3QiLCJwZXJtIjoic2lnbiIsImV4dCI6IiJ9.FEPMm5aKcm7pyn7iDMRl4CEs0-X3MQpgjORPRy9WPso")
	pvc := &ProofEventClient{}
	closer, err := jsonrpc.NewMergeClient(ctx, "ws://127.0.0.1:45132/rpc/v0", "Filecoin", []interface{}{pvc}, headers)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer closer()

	actorAddr, _ := address.NewIDAddress(7)
	result, err := pvc.ComputeProof(ctx, actorAddr, []proof5.SectorInfo{{
		SealProof:    1,
		SectorNumber: 0,
		SealedCID:    cid.Undef,
	}}, []byte{1, 2, 3})
	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Println(result)
}

type WalletEventClient struct {
	WalletHas  func(ctx context.Context, supportAccount string, addr address.Address) (bool, error)
	WalletSign func(ctx context.Context, account string, addr address.Address, toSign []byte, meta wallet.MsgMeta) (*crypto.Signature, error)
}

func WalletHas() {
	ctx := context.Background()
	headers := http.Header{}
	headers.Add("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoic3Rlc3QiLCJwZXJtIjoic2lnbiIsImV4dCI6IiJ9.FEPMm5aKcm7pyn7iDMRl4CEs0-X3MQpgjORPRy9WPso")
	pvc := &WalletEventClient{}
	closer, err := jsonrpc.NewMergeClient(ctx, "ws://127.0.0.1:45132/rpc/v0", "Filecoin", []interface{}{pvc}, headers)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer closer()

	actorAddr, _ := address.NewIDAddress(1)
	result, err := pvc.WalletHas(ctx, "stest", actorAddr)
	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Println(result)

	result, err = pvc.WalletHas(ctx, "wtest2", actorAddr)
	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Println(result)

	actorAddr2, _ := address.NewIDAddress(8)
	result, err = pvc.WalletHas(ctx, "wtest2", actorAddr2)
	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Println(result)
}

func WalletSign() {
	ctx := context.Background()
	headers := http.Header{}
	headers.Add("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoic3Rlc3QiLCJwZXJtIjoic2lnbiIsImV4dCI6IiJ9.FEPMm5aKcm7pyn7iDMRl4CEs0-X3MQpgjORPRy9WPso")
	pvc := &WalletEventClient{}
	closer, err := jsonrpc.NewMergeClient(ctx, "ws://127.0.0.1:45132/rpc/v0", "Filecoin", []interface{}{pvc}, headers)
	if err != nil {
		log.Fatal(err)
		return
	}
	defer closer()

	actorAddr, _ := address.NewIDAddress(7)
	result, err := pvc.WalletSign(ctx, "wtest", actorAddr, []byte{1, 2}, wallet.MsgMeta{
		Type:  wallet.MTUnknown,
		Extra: nil,
	})
	if err != nil {
		log.Fatal(err)
		return
	}

	fmt.Println(result)
}
