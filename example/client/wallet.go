package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types "github.com/filecoin-project/venus/venus-shared/types/gateway"
)

type WalletEventClient struct {
	ResponseWalletEvent func(ctx context.Context, resp *types.ResponseEvent) error
	ListenWalletEvent   func(ctx context.Context, policy *types.WalletRegisterPolicy) (chan *types.RequestEvent, error)
	SupportNewAccount   func(ctx context.Context, channelId sharedTypes.UUID, account string) error
}

func main() {
	/*for i := 0; i < 1; i++ {
		go func() {
			fmt.Println("NewWalletClient")
			NewWalletClient()
		}()
	}*/
	for i := 0; i < 1; i++ {
		go func() {
			fmt.Println("NewProofClient")
			NewProofClient()
		}()
	}
	ch := make(chan struct{})
	<-ch
}

var token = "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiR2F0ZVdheUxvY2FsVG9rZW4iLCJwZXJtIjoiYWRtaW4iLCJleHQiOiIifQ.jZOlCBnxZtwc9PsjY7OMnooK6C3PFExvZesWsFrVyCs"

// nolint
func NewWalletClient() jsonrpc.ClientCloser {
	ctx := context.Background()
	pvc := &WalletEventClient{}
	headers := http.Header{}
	headers.Add("Authorization", token)
	closer, err := jsonrpc.NewMergeClient(ctx, "ws://127.0.0.1:45132/rpc/v0", "Gateway", []interface{}{pvc}, headers)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	eventCh, err := pvc.ListenWalletEvent(ctx, &types.WalletRegisterPolicy{SupportAccounts: []string{"test_user"}})
	if err != nil {
		log.Fatal(err)
		return nil
	}
	var channel sharedTypes.UUID

	cc := make(chan struct{})
	go func() {
		<-cc
		_ = pvc.SupportNewAccount(ctx, channel, "stest")
	}()

	for event := range eventCh {
		switch event.Method {
		case "InitConnect":
			req := types.ConnectedCompleted{}
			err := json.Unmarshal(event.Payload, &req)
			if err != nil {
				_ = pvc.ResponseWalletEvent(ctx, &types.ResponseEvent{
					ID:      event.ID,
					Payload: nil,
					Error:   err.Error(),
				})
			}
			channel = req.ChannelId
			cc <- struct{}{}
		case "WalletList":
			fmt.Println("receive wallet list req")
			addr1, _ := address.NewIDAddress(1)
			addrBytes, _ := json.Marshal([]address.Address{addr1})
			_ = pvc.ResponseWalletEvent(ctx, &types.ResponseEvent{
				ID:      event.ID,
				Payload: addrBytes,
				//		Error:   err.Error(),
			})
		case "WalletSign":
			req := types.WalletSignRequest{}
			_ = json.Unmarshal(event.Payload, &req)
			fmt.Println("address", req.Signer)
			fmt.Println("tosign", req.ToSign)
			_ = pvc.ResponseWalletEvent(ctx, &types.ResponseEvent{
				ID:      event.ID,
				Payload: []byte{1, 2, 3, 54, 6},
				//		Error:   err.Error(),
			})
		}
	}
	return closer
}
