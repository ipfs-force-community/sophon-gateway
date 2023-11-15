package walletevent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"

	"github.com/filecoin-project/venus/venus-shared/api"
	v2API "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	sharedGatewayTypes "github.com/filecoin-project/venus/venus-shared/types/gateway"

	"github.com/ipfs-force-community/sophon-gateway/types"
)

func NewWalletRegisterClient(ctx context.Context, url, token string) (v2API.IWalletServiceProvider, jsonrpc.ClientCloser, error) {
	headers := http.Header{}
	headers.Add(api.AuthorizationHeader, "Bearer "+token)
	client, closer, err := v2API.NewIGatewayRPC(ctx, url, headers)
	if err != nil {
		return nil, nil, err
	}
	return client, closer, nil
}

type WalletEventClient struct {
	processor          types.IWalletHandler
	client             v2API.IWalletServiceProvider
	randomBytes        []byte
	log                *zap.SugaredLogger
	channel            sharedTypes.UUID
	getSupportAccounts func() []string
	readyCh            chan struct{}
}

func NewWalletEventClient(ctx context.Context, process types.IWalletHandler, client v2API.IWalletServiceProvider, log *zap.SugaredLogger, getSupportAccounts func() []string) *WalletEventClient {
	return &WalletEventClient{
		processor:          process,
		client:             client,
		log:                log,
		getSupportAccounts: getSupportAccounts,
		randomBytes:        sharedGatewayTypes.RandomBytes,
		readyCh:            make(chan struct{}, 1),
	}
}

func (e *WalletEventClient) SupportAccount(ctx context.Context, supportAccount string) error {
	err := e.client.SupportNewAccount(ctx, e.channel, supportAccount)
	if err != nil {
		return err
	}
	return nil
}

func (e *WalletEventClient) AddNewAddress(ctx context.Context, newAddrs []address.Address) error {
	return e.client.AddNewAddress(ctx, e.channel, newAddrs)
}

func (e *WalletEventClient) RemoveAddress(ctx context.Context, newAddrs []address.Address) error {
	return e.client.RemoveAddress(ctx, e.channel, newAddrs)
}

func (e *WalletEventClient) ListenWalletRequest(ctx context.Context) {
	for {
		if err := e.listenWalletRequestOnce(ctx); err != nil {
			e.log.Errorf("listen wallet event errored: %s", err)
		} else {
			e.log.Warn("listenWalletRequestOnce quit, try again")
		}
		select {
		case <-time.After(time.Second):
		case <-ctx.Done():
			e.log.Warnf("not restarting listenWalletRequestOnce: context error: %s", ctx.Err())
			return
		}
		e.log.Info("restarting listenWalletRequestOnce")
		// try clear ready channel
		select {
		case <-e.readyCh:
		default:
		}
	}
}

func (e *WalletEventClient) WaitReady(ctx context.Context) {
	select {
	case <-e.readyCh:
	case <-ctx.Done():
	}
}

func (e *WalletEventClient) listenWalletRequestOnce(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	accounts := e.getSupportAccounts()
	policy := &sharedGatewayTypes.WalletRegisterPolicy{
		SupportAccounts: accounts,
		SignBytes:       e.randomBytes,
	}
	e.log.Infow("", "rand sign byte", e.randomBytes, "support accounts", accounts)
	walletEventCh, err := e.client.ListenWalletEvent(ctx, policy)
	if err != nil {
		// Retry is handled by caller
		return fmt.Errorf("listenWalletRequestOnce listenWalletRequestOnce call failed: %w", err)
	}

	for event := range walletEventCh {
		switch event.Method {
		case "InitConnect":
			req := sharedGatewayTypes.ConnectedCompleted{}
			err := json.Unmarshal(event.Payload, &req)
			if err != nil {
				e.log.Errorf("init connect error %s", err)
			}
			e.channel = req.ChannelId
			e.log.Infof("connect to server success %v", req.ChannelId)
			e.readyCh <- struct{}{}
			// do not response
		case "WalletList":
			go e.walletList(ctx, event.ID)
		case "WalletSign":
			go e.walletSign(ctx, event)
		default:
			e.log.Errorf("unexpect proof event type %s", event.Method)
		}
	}

	return nil
}

func (e *WalletEventClient) walletList(ctx context.Context, id sharedTypes.UUID) {
	addrs, err := e.processor.WalletList(ctx)
	if err != nil {
		e.log.Errorf("WalletList error %s", err)
		e.error(ctx, id, err)
		return
	}
	e.value(ctx, id, addrs)
}

func (e *WalletEventClient) walletSign(ctx context.Context, event *sharedGatewayTypes.RequestEvent) {
	e.log.Debug("receive WalletSign event")
	req := sharedGatewayTypes.WalletSignRequest{}
	err := json.Unmarshal(event.Payload, &req)
	if err != nil {
		e.log.Errorf("unmarshal WalletSignRequest error %s", err)
		e.error(ctx, event.ID, err)
		return
	}
	e.log.Debug("start WalletSign")
	sig, err := e.processor.WalletSign(ctx, req.Signer, req.ToSign, sharedTypes.MsgMeta{Type: req.Meta.Type, Extra: req.Meta.Extra})
	if err != nil {
		e.log.Errorf("WalletSign error %s", err)
		e.error(ctx, event.ID, err)
		return
	}
	e.log.Debug("end WalletSign")
	e.value(ctx, event.ID, sig)
	e.log.Debug("end WalletSign response")
}

func (e *WalletEventClient) value(ctx context.Context, id sharedTypes.UUID, val interface{}) {
	respBytes, err := json.Marshal(val)
	if err != nil {
		e.log.Errorf("marshal address list error %s", err)
		err = e.client.ResponseWalletEvent(ctx, &sharedGatewayTypes.ResponseEvent{
			ID:      id,
			Payload: nil,
			Error:   err.Error(),
		})
		e.log.Errorf("response wallet event error %s", err)
		return
	}
	err = e.client.ResponseWalletEvent(ctx, &sharedGatewayTypes.ResponseEvent{
		ID:      id,
		Payload: respBytes,
		Error:   "",
	})
	if err != nil {
		e.log.Errorf("response error %v", err)
	}
}

func (e *WalletEventClient) error(ctx context.Context, id sharedTypes.UUID, err error) {
	err = e.client.ResponseWalletEvent(ctx, &sharedGatewayTypes.ResponseEvent{
		ID:      id,
		Payload: nil,
		Error:   err.Error(),
	})
	if err != nil {
		e.log.Errorf("response error %v", err)
	}
}
