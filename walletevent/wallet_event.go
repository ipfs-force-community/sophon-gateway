package walletevent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/venus-wallet/core"
	"github.com/google/uuid"
	"github.com/ipfs-force-community/venus-gateway/types"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"
	"sync"
	"time"
)

var log = logging.Logger("event_stream")

var _ IWalletEvent = (*WalletEventStream)(nil)

type WalletEventStream struct {
	reqLk     sync.RWMutex
	idRequest map[uuid.UUID]*types.RequestEvent

	walletConnMgr IWalletConnMgr
	cfg           *types.Config
}

func NewWalletEventStream(cfg *types.Config) *WalletEventStream {
	return &WalletEventStream{
		reqLk:         sync.RWMutex{},
		idRequest:     make(map[uuid.UUID]*types.RequestEvent),
		walletConnMgr: newWalletConnMgr(),
		cfg:           cfg,
	}
}

func (e *WalletEventStream) ListenWalletEvent(ctx context.Context, supportAccounts []string) (chan *types.RequestEvent, error) {
	walletAccount := ctx.Value(types.AccountKey).(string)
	out := make(chan *types.RequestEvent, e.cfg.RequestQueueSize)

	go func() {
		//todo validate the account exit or not
		addrs, err := e.getValidatedAddress(ctx, out)
		if err != nil {
			close(out)
			log.Errorf("validate address error %v", err)
			return
		}
		fmt.Println("scan address", addrs)
		walletChannelInfo := newWalletChannelInfo(types.NewChannelInfo(out), addrs)

		err = e.walletConnMgr.AddNewConn(walletAccount, supportAccounts, addrs, walletChannelInfo)
		if err != nil {
			close(out)
			log.Errorf("validate address error %v", err)
			return
		}

		log.Infof("add new connections %s", walletChannelInfo.ChannelId)
		//todo rescan address to add new address or remove

		connectBytes, err := json.Marshal(types.ConnectedCompleted{
			ChannelId: walletChannelInfo.ChannelId,
		})
		if err != nil {
			close(out)
			log.Errorf("marshal failed %v", err)
			return
		}

		out <- &types.RequestEvent{
			Id:         uuid.New(),
			Method:     "InitConnect",
			CreateTime: time.Now(),
			Payload:    connectBytes,
			Result:     nil,
		} //not response

		for {
			select {
			case <-ctx.Done():
				err := e.walletConnMgr.RemoveConn(walletAccount, walletChannelInfo)
				if err != nil {
					log.Errorf("validate address error %v", err)
				}
				close(out)
				return
			}
		}
	}()
	return out, nil
}

func (e *WalletEventStream) SupportNewAccount(ctx context.Context, channelId, account string) error {
	walletAccount := ctx.Value(types.AccountKey).(string)
	return e.walletConnMgr.AddSupportAccount(walletAccount, account)
}

func (e *WalletEventStream) ResponseEvent(ctx context.Context, resp *types.ResponseEvent) error {
	e.reqLk.Lock()
	event, ok := e.idRequest[resp.Id]
	if ok {
		delete(e.idRequest, resp.Id)
	} else {
		log.Errorf("id request not exit %v", resp)
	}
	e.reqLk.Unlock()
	if ok {
		event.Result <- resp
	}
	return nil
}

func (e *WalletEventStream) sendRequest(ctx context.Context, req *walletPayloadRequest) error {
	//select connections
	conn, err := e.walletConnMgr.GetChannel(req.Account, req.Addr)
	if err != nil {
		return xerrors.Errorf("cannot find any connection address %s, account %s", req.Addr, req.Account)
	}

	id := uuid.New()
	request := &types.RequestEvent{
		Id:         id,
		Method:     req.Method,
		Payload:    req.Payload,
		CreateTime: time.Now(),
		Result:     req.Result,
	}
	e.reqLk.Lock()
	e.idRequest[id] = request
	e.reqLk.Unlock()
	//timeout here
	select {
	case conn.OutBound <- request:
	case <-ctx.Done():
		return xerrors.Errorf("send request cancel by context")
	case <-time.After(e.cfg.RequestTimeout):
		e.reqLk.Lock()
		delete(e.idRequest, id)
		e.reqLk.Unlock()
		return xerrors.Errorf("request %s too long not response", id)
	}

	return nil
}

func (e *WalletEventStream) WalletHas(ctx context.Context, supportAccount string, addr address.Address) (bool, error) {
	return e.walletConnMgr.HasWalletChannel(supportAccount, addr)
}

func (e *WalletEventStream) WalletSign(ctx context.Context, account string, addr address.Address, toSign []byte, meta core.MsgMeta) (*crypto.Signature, error) {
	payload, err := json.Marshal(&types.WalletSignRequest{
		Signer: addr,
		ToSign: toSign,
		Meta:   meta,
	})
	if err != nil {
		return nil, err
	}

	resultCh := make(chan *types.ResponseEvent)
	req := &walletPayloadRequest{
		Account: account,
		Addr:    addr,
		Method:  "WalletSign",
		Payload: payload,
		Result:  resultCh,
	}

	err = e.sendRequest(ctx, req)
	if err != nil {
		return nil, err
	}

	respEvent := <-resultCh
	if len(respEvent.Error) > 0 {
		return nil, errors.New(respEvent.Error)
	}
	var has crypto.Signature
	err = json.Unmarshal(respEvent.Payload, &has)
	if err != nil {
		return nil, err
	}
	return &has, nil
}

func (e *WalletEventStream) ListWalletInfo(ctx context.Context) ([]*WalletDetail, error) {
	return e.walletConnMgr.ListWalletInfo(ctx)
}

func (e *WalletEventStream) ListWalletInfoByWallet(ctx context.Context, wallet string) (*WalletDetail, error) {
	return e.walletConnMgr.ListWalletInfoByWallet(ctx, wallet)
}

func (e *WalletEventStream) getValidatedAddress(ctx context.Context, out chan *types.RequestEvent) ([]address.Address, error) {
	id := uuid.New()
	resultCh := make(chan *types.ResponseEvent)
	req := &types.RequestEvent{
		Id:         id,
		Method:     "WalletList",
		Payload:    nil,
		CreateTime: time.Now(),
		Result:     resultCh,
	}

	e.reqLk.Lock()
	e.idRequest[id] = req
	e.reqLk.Unlock()
	out <- req

	respEvent := <-resultCh
	if len(respEvent.Error) > 0 {
		return nil, errors.New(respEvent.Error)
	}
	var result []address.Address
	err := json.Unmarshal(respEvent.Payload, &result)
	if err != nil {
		return nil, err
	}
	//todo validate the wallet is really has the address

	return result, nil
}

func (e *WalletEventStream) validateAddress(ctx context.Context, addr address.Address, out chan *types.RequestEvent) (bool, error) {
	id := uuid.New()
	resultCh := make(chan *types.ResponseEvent)
	req := &types.RequestEvent{
		Id:         id,
		Method:     "WalletValidate",
		Payload:    nil,
		CreateTime: time.Now(),
		Result:     resultCh,
	}

	e.reqLk.Lock()
	e.idRequest[id] = req
	e.reqLk.Unlock()
	out <- req

	respEvent := <-resultCh
	if len(respEvent.Error) > 0 {
		return false, errors.New(respEvent.Error)
	}
	var result bool
	err := json.Unmarshal(respEvent.Payload, &result)
	if err != nil {
		return false, err
	}
	return result, nil
}
