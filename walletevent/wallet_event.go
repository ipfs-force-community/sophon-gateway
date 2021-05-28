package walletevent

import (
	"context"
	"encoding/json"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/venus-wallet/core"
	"github.com/google/uuid"
	"github.com/ipfs-force-community/venus-gateway/types"
	logging "github.com/ipfs/go-log/v2"
	"time"
)

var log = logging.Logger("event_stream")

var _ IWalletEvent = (*WalletEventStream)(nil)

type WalletEventStream struct {
	walletConnMgr IWalletConnMgr
	*types.BaseEventStream
	cfg *types.Config
}

func NewWalletEventStream(ctx context.Context, cfg *types.Config) *WalletEventStream {
	walletEventStream := &WalletEventStream{
		walletConnMgr:   newWalletConnMgr(),
		BaseEventStream: types.NewBaseEventStream(ctx, cfg),
		cfg:             cfg,
	}
	return walletEventStream
}

func (e *WalletEventStream) ListenWalletEvent(ctx context.Context, policy *WalletRegisterPolicy) (chan *types.RequestEvent, error) {
	walletAccount := ctx.Value(types.AccountKey).(string)
	ip := ctx.Value(types.IPKey).(string)
	out := make(chan *types.RequestEvent, e.cfg.RequestQueueSize)

	go func() {
		channel := types.NewChannelInfo(ip, out)
		//todo validate the account exit or not
		addrs, err := e.getValidatedAddress(ctx, channel)
		if err != nil {
			close(out)
			log.Errorf("validate address error %v", err)
			return
		}

		walletChannelInfo := newWalletChannelInfo(channel, addrs)

		err = e.walletConnMgr.AddNewConn(walletAccount, policy.SupportAccounts, addrs, walletChannelInfo)
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

	channels, err := e.walletConnMgr.GetChannels(account, addr)
	if err != nil {
		return nil, err
	}

	var result crypto.Signature
	err = e.SendRequest(ctx, channels, "WalletSign", payload, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (e *WalletEventStream) ListWalletInfo(ctx context.Context) ([]*WalletDetail, error) {
	return e.walletConnMgr.ListWalletInfo(ctx)
}

func (e *WalletEventStream) ListWalletInfoByWallet(ctx context.Context, wallet string) (*WalletDetail, error) {
	return e.walletConnMgr.ListWalletInfoByWallet(ctx, wallet)
}

func (e *WalletEventStream) getValidatedAddress(ctx context.Context, channel *types.ChannelInfo) ([]address.Address, error) {
	var result []address.Address
	err := e.SendRequest(ctx, []*types.ChannelInfo{channel}, "WalletList", nil, &result)
	if err != nil {
		return nil, err
	}
	//todo validate the wallet is really has the address
	return result, nil
}

func (e *WalletEventStream) validateAddress(ctx context.Context, addr address.Address, channel *types.ChannelInfo) (bool, error) {
	var result bool
	err := e.SendRequest(ctx, []*types.ChannelInfo{channel}, "WalletValidate", nil, &result)
	if err != nil {
		return false, err
	}
	return result, nil
}
