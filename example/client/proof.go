package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"

	proof5 "github.com/filecoin-project/specs-actors/v5/actors/runtime/proof"

	types "github.com/filecoin-project/venus/venus-shared/types/gateway"
)

// ProofEventClient test proof event client
type ProofEventClient struct {
	ResponseProofEvent func(ctx context.Context, resp *types.ResponseEvent) error
	ListenProofEvent   func(ctx context.Context, policy *types.ProofRegisterPolicy) (chan *types.RequestEvent, error)
}

// NewProofClient create test client
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
			closer()
			continue
		}

		// rand.Seed(time.Now().Unix())
		actorAddr, _ := address.NewIDAddress(7)
		eventCh, err := pvc.ListenProofEvent(ctx, &types.ProofRegisterPolicy{MinerAddress: actorAddr})
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
					_ = pvc.ResponseProofEvent(ctx, &types.ResponseEvent{
						ID:      event.ID,
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
				_ = pvc.ResponseProofEvent(ctx, &types.ResponseEvent{
					ID:      event.ID,
					Payload: proofBytes,
					Error:   "",
				})
			}
		}
		closer()
	}
}
