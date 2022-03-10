package walletevent

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"io"
	"io/ioutil"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/venus-auth/cmd/jwtclient"
	wcrypto "github.com/filecoin-project/venus/pkg/crypto"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types2 "github.com/filecoin-project/venus/venus-shared/types/gateway"
	"github.com/ipfs-force-community/venus-gateway/types"
	logging "github.com/ipfs/go-log/v2"
	"golang.org/x/xerrors"
)

var log = logging.Logger("event_stream")

var _ IWalletEvent = (*WalletEventStream)(nil)

var hash256 = sha256.New()

type WalletEventStream struct {
	walletConnMgr IWalletConnMgr
	cfg           *types.Config
	authClient    types.IAuthClient
	randBytes     []byte
	*types.BaseEventStream
}

func NewWalletEventStream(ctx context.Context, authClient types.IAuthClient, cfg *types.Config) *WalletEventStream {
	walletEventStream := &WalletEventStream{
		walletConnMgr:   newWalletConnMgr(),
		BaseEventStream: types.NewBaseEventStream(ctx, cfg),
		cfg:             cfg,
		authClient:      authClient,
	}
	var err error
	walletEventStream.randBytes, err = ioutil.ReadAll(io.LimitReader(rand.Reader, 32))
	if err != nil {
		panic(xerrors.Errorf("rand secret failed %v", err))
	}
	log.Infow("", "rand secret", walletEventStream.randBytes)
	return walletEventStream
}

func (e *WalletEventStream) ListenWalletEvent(ctx context.Context, policy *types2.WalletRegisterPolicy) (chan *types2.RequestEvent, error) {
	walletAccount, _ := jwtclient.CtxGetName(ctx)
	ip, _ := jwtclient.CtxGetTokenLocation(ctx)
	out := make(chan *types2.RequestEvent, e.cfg.RequestQueueSize)

	go func() {
		channel := types.NewChannelInfo(ip, out)
		//todo validate the account exit or not
		addrs, err := e.getValidatedAddress(ctx, channel, policy.SignBytes, walletAccount)
		if err != nil {
			close(out)
			log.Error(err)
			return
		}

		walletChannelInfo := newWalletChannelInfo(channel, addrs, policy.SignBytes)

		err = e.walletConnMgr.AddNewConn(walletAccount, policy, addrs, walletChannelInfo)
		if err != nil {
			close(out)
			log.Errorf("validate address error %v", err)
			return
		}

		log.Infof("add new connections %s %s", walletAccount, walletChannelInfo.ChannelId)
		//todo rescan address to add new address or remove

		connectBytes, err := json.Marshal(types2.ConnectedCompleted{
			ChannelId: walletChannelInfo.ChannelId,
		})
		if err != nil {
			close(out)
			log.Errorf("marshal failed %v", err)
			return
		}

		out <- &types2.RequestEvent{
			ID:         sharedTypes.NewUUID(),
			Method:     "InitConnect",
			CreateTime: time.Now(),
			Payload:    connectBytes,
			Result:     nil,
		} //not response

		<-ctx.Done()
		if e.walletConnMgr.RemoveConn(walletAccount, walletChannelInfo); err != nil {
			log.Errorf("validate address error %v", err)
		}
		close(out)
	}()
	return out, nil
}

func (e *WalletEventStream) SupportNewAccount(ctx context.Context, channelId sharedTypes.UUID, account string) error {
	walletAccount, _ := jwtclient.CtxGetName(ctx)
	err := e.walletConnMgr.AddSupportAccount(walletAccount, account)
	if err == nil {
		log.Infof("wallet %s add account %s", walletAccount, account)
	} else {
		log.Errorf("wallet %s add account %s failed %v", walletAccount, account, err)
	}

	return err
}

func (e *WalletEventStream) AddNewAddress(ctx context.Context, channelId sharedTypes.UUID, addrs []address.Address) error {
	walletAccount, _ := jwtclient.CtxGetName(ctx)
	info, err := e.walletConnMgr.GetConn(walletAccount, channelId)
	if err != nil {
		return err
	}
	for _, addr := range addrs {
		if err := e.verifyAddress(ctx, addr, info.ChannelInfo, info.signBytes, walletAccount); err != nil {
			return err
		}
	}
	if err = e.walletConnMgr.NewAddress(walletAccount, channelId, addrs); err == nil {
		log.Infof("wallet %s add address %v", walletAccount, addrs)
	} else {
		log.Errorf("wallet %s add address %v failed %v", walletAccount, addrs, err)
	}

	return err
}

func (e *WalletEventStream) RemoveAddress(ctx context.Context, channelId sharedTypes.UUID, addrs []address.Address) error {
	walletAccount, _ := jwtclient.CtxGetName(ctx)
	err := e.walletConnMgr.RemoveAddress(walletAccount, channelId, addrs)
	if err == nil {
		log.Infof("wallet %s remove address %v", walletAccount, addrs)
	} else {
		log.Infof("wallet %s remove address %v failed %v", walletAccount, addrs, err)
	}
	return err
}

func (e *WalletEventStream) WalletHas(ctx context.Context, supportAccount string, addr address.Address) (bool, error) {
	return e.walletConnMgr.HasWalletChannel(supportAccount, addr)
}

func (e *WalletEventStream) WalletSign(ctx context.Context, account string, addr address.Address, toSign []byte, meta sharedTypes.MsgMeta) (*crypto.Signature, error) {
	payload, err := json.Marshal(&types2.WalletSignRequest{
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

func (e *WalletEventStream) ListWalletInfo(ctx context.Context) ([]*types2.WalletDetail, error) {
	return e.walletConnMgr.ListWalletInfo(ctx)
}

func (e *WalletEventStream) ListWalletInfoByWallet(ctx context.Context, wallet string) (*types2.WalletDetail, error) {
	return e.walletConnMgr.ListWalletInfoByWallet(ctx, wallet)
}

func (e *WalletEventStream) getValidatedAddress(ctx context.Context, channel *types.ChannelInfo, signBytes []byte, walletAccount string) ([]address.Address, error) {
	var addrs []address.Address
	err := e.SendRequest(ctx, []*types.ChannelInfo{channel}, "WalletList", nil, &addrs)
	if err != nil {
		return nil, err
	}

	// validate the wallet is really has the address
	validAddrs := make([]address.Address, 0, len(addrs))
	for _, addr := range addrs {
		if err := e.verifyAddress(ctx, addr, channel, signBytes, walletAccount); err != nil {
			return nil, err
		}
		validAddrs = append(validAddrs, addr)
	}

	return validAddrs, nil
}

func (e *WalletEventStream) verifyAddress(ctx context.Context, addr address.Address, channel *types.ChannelInfo, signBytes []byte, walletAccount string) error {
	hasher := sha256.New()
	_, _ = hasher.Write(append(e.randBytes, signBytes...))
	signData := hash256.Sum(nil)
	payload, err := json.Marshal(&types2.WalletSignRequest{
		Signer: addr,
		ToSign: signData,
		Meta:   sharedTypes.MsgMeta{Type: sharedTypes.MTVerifyAddress, Extra: e.randBytes},
	})
	if err != nil {
		return err
	}
	var sig crypto.Signature
	err = e.SendRequest(ctx, []*types.ChannelInfo{channel}, "WalletSign", payload, &sig)
	if err != nil {
		return xerrors.Errorf("wallet %s verify address %s failed, signed error %v", walletAccount, addr.String(), err)
	}
	if err := wcrypto.Verify(&sig, addr, signData); err != nil {
		return xerrors.Errorf("wallet %s verify address %s failed: %v", walletAccount, addr.String(), err)
	}
	log.Infof("wallet %s verify address %s success", walletAccount, addr)

	return nil
}
