package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/google/uuid"
	"github.com/ipfs-force-community/venus-gateway/types"
	"log"
	"net/http"
)

type WalletEventClient struct {
	ResponseWalletEvent func(ctx context.Context, resp *types.ResponseEvent) error
	ListenWalletEvent   func(ctx context.Context, supportAccounts []string) (chan *types.RequestEvent, error)
	SupportNewAccount   func(ctx context.Context, channelId string, account string) error
}

func main() {
	for i := 0; i < 10; i++ {
		go func() {
			fmt.Println("NewWalletClient")
			NewWalletClient()
		}()
	}
	for i := 0; i < 10; i++ {

		go func() {
			fmt.Println("NewProofClient")
			NewProofClient()
		}()
	}
	ch := make(chan struct{})
	<-ch
}

func NewWalletClient() jsonrpc.ClientCloser {
	ctx := context.Background()
	pvc := &WalletEventClient{}
	headers := http.Header{}
	headers.Add("Authorization", "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoid3Rlc3QiLCJwZXJtIjoic2lnbiIsImV4dCI6IiJ9.Tkc9iwsT5k_UQakD9RJl5azKRm9Fzs_AkJXitEW5Krk")
	closer, err := jsonrpc.NewMergeClient(ctx, "ws://127.0.0.1:45132/rpc/v0", "Filecoin", []interface{}{pvc}, headers)
	if err != nil {
		log.Fatal(err)
		return nil
	}

	eventCh, err := pvc.ListenWalletEvent(ctx, []string{"test"})
	if err != nil {
		log.Fatal(err)
		return nil
	}
	var channel uuid.UUID

	cc := make(chan struct{})
	go func() {
		<-cc
		pvc.SupportNewAccount(ctx, channel.String(), "stest")
	}()

	for event := range eventCh {
		switch event.Method {
		case "InitConnect":
			req := types.ConnectedCompleted{}
			err := json.Unmarshal(event.Payload, &req)
			if err != nil {
				pvc.ResponseWalletEvent(ctx, &types.ResponseEvent{
					Id:      event.Id,
					Payload: nil,
					Error:   err.Error(),
				})
			}
			channel = req.ChannelId
			cc <- struct{}{}
		case "WalletList":
			fmt.Println("receive wallet list req")
			addr1, _ := address.NewIDAddress(1)
			xxxx, _ := json.Marshal([]address.Address{addr1})
			pvc.ResponseWalletEvent(ctx, &types.ResponseEvent{
				Id:      event.Id,
				Payload: xxxx,
				//		Error:   err.Error(),
			})
		case "WalletSign":
			req := types.WalletSignRequest{}
			json.Unmarshal(event.Payload, &req)
			fmt.Println("address", req.Signer)
			fmt.Println("tosign", req.ToSign)
			pvc.ResponseWalletEvent(ctx, &types.ResponseEvent{
				Id:      event.Id,
				Payload: []byte{1, 2, 3, 54, 6},
				//		Error:   err.Error(),
			})
		}
	}
	return closer
}
