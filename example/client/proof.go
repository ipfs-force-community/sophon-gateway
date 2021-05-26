package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/specs-actors/actors/runtime/proof"
	"github.com/ipfs-force-community/venus-gateway/types"
	"log"
	"net/http"
	"time"
)

type ProofEventClient struct {
	ResponseProofEvent func(ctx context.Context, resp *types.ResponseEvent) error
	ListenProofEvent   func(ctx context.Context, mAddr address.Address) (chan *types.RequestEvent, error)
}

func NewProofClient() {
	for {
		time.Sleep(time.Second)
		ctx := context.Background()
		headers := http.Header{}
		headers.Add("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoic3Rlc3QiLCJwZXJtIjoic2lnbiIsImV4dCI6IiJ9.FEPMm5aKcm7pyn7iDMRl4CEs0-X3MQpgjORPRy9WPso")
		pvc := &ProofEventClient{}
		closer, err := jsonrpc.NewMergeClient(ctx, "ws://127.0.0.1:45132/rpc/v0", "Filecoin", []interface{}{pvc}, headers)
		if err != nil {
			log.Fatal(err)
			continue
		}
		defer closer()

		//rand.Seed(time.Now().Unix())
		actorAddr, _ := address.NewIDAddress(7)
		eventCh, err := pvc.ListenProofEvent(ctx, actorAddr)
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

				proof := []proof.PoStProof{
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
