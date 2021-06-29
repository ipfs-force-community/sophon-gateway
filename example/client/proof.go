package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ipfs-force-community/venus-gateway/proofevent"
	"log"
	"net/http"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"

	proof5 "github.com/filecoin-project/specs-actors/v5/actors/runtime/proof"

	"github.com/ipfs-force-community/venus-gateway/types"
)

type ProofEventClient struct {
	ResponseProofEvent func(ctx context.Context, resp *types.ResponseEvent) error
	ListenProofEvent   func(ctx context.Context, policy *proofevent.ProofRegisterPolicy) (chan *types.RequestEvent, error)
}

func NewProofClient() {
	for {
		time.Sleep(time.Second)
		ctx := context.Background()
		headers := http.Header{}
		headers.Add("Authorization", token)
		pvc := &ProofEventClient{}
		closer, err := jsonrpc.NewMergeClient(ctx, "ws://127.0.0.1:45132/rpc/v0", "Gateway", []interface{}{pvc}, headers)
		if err != nil {
			log.Fatal(err)
			continue
		}
		defer closer()

		// rand.Seed(time.Now().Unix())
		actorAddr, _ := address.NewIDAddress(7)
		eventCh, err := pvc.ListenProofEvent(ctx, &proofevent.ProofRegisterPolicy{MinerAddress: actorAddr})
		if err != nil {
			log.Fatal(err)
			continue
		}
		for event := range eventCh {
			switch event.Method {
			case "ComputeProof":
				req := types.ComputeProofRequest{}
				err := json.Unmarshal(event.Payload, &req)
				if err != nil {
					fmt.Println(event.Id.String())
					pvc.ResponseProofEvent(ctx, &types.ResponseEvent{
						Id:      event.Id,
						Payload: nil,
						Error:   err.Error(),
					})
					continue
				}

				proof := []proof5.PoStProof{
					{
						PoStProof:  0,
						ProofBytes: []byte{2, 3, 4},
					},
				}
				proofBytes, _ := json.Marshal(proof)
				pvc.ResponseProofEvent(ctx, &types.ResponseEvent{
					Id:      event.Id,
					Payload: proofBytes,
					Error:   "",
				})
			}
		}
	}
}
