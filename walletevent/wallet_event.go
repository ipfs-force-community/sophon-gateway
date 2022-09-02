package walletevent

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"time"

	"go.opencensus.io/stats"
	"go.opencensus.io/tag"

	logging "github.com/ipfs/go-log/v2"
	"github.com/pkg/errors"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/crypto"

	"github.com/filecoin-project/venus-auth/auth"
	"github.com/filecoin-project/venus-auth/jwtclient"

	wcrypto "github.com/filecoin-project/venus/pkg/crypto"
	"github.com/filecoin-project/venus/venus-shared/api/gateway/v1"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types2 "github.com/filecoin-project/venus/venus-shared/types/gateway"

	"github.com/ipfs-force-community/venus-gateway/metrics"
	"github.com/ipfs-force-community/venus-gateway/types"
)

var log = logging.Logger("event_stream")

var _ gateway.IWalletClient = (*WalletEventStream)(nil)

type WalletEventStream struct {
	walletConnMgr IWalletConnMgr
	cfg           *types.RequestConfig
	authClient    types.IAuthClient
	randBytes     []byte
	*types.BaseEventStream

	disableVerifyWalletAddrs bool
}

func NewWalletEventStream(ctx context.Context, authClient types.IAuthClient, cfg *types.RequestConfig, diableVerifyWalletAddrs bool) *WalletEventStream {
	walletEventStream := &WalletEventStream{
		walletConnMgr:            newWalletConnMgr(),
		BaseEventStream:          types.NewBaseEventStream(ctx, cfg),
		cfg:                      cfg,
		authClient:               authClient,
		disableVerifyWalletAddrs: diableVerifyWalletAddrs,
	}
	var err error
	walletEventStream.randBytes, err = ioutil.ReadAll(io.LimitReader(rand.Reader, 32))
	if err != nil {
		panic(fmt.Errorf("rand secret failed %v", err))
	}
	log.Infow("", "rand secret", walletEventStream.randBytes)
	return walletEventStream
}

func (w *WalletEventStream) ListenWalletEvent(ctx context.Context, policy *types2.WalletRegisterPolicy) (<-chan *types2.RequestEvent, error) {
	walletAccount, exit := jwtclient.CtxGetName(ctx)
	if !exit {
		return nil, errors.New("unable to get account name in method ListenWalletEvent request")
	}

	ip, _ := jwtclient.CtxGetTokenLocation(ctx) //todo sure exit?
	out := make(chan *types2.RequestEvent, w.cfg.RequestQueueSize)

	ctx, _ = tag.New(ctx, tag.Upsert(metrics.WalletAccountKey, walletAccount), tag.Upsert(metrics.IPKey, ip))

	// Account must exist in venus-auth
	_, err := w.authClient.GetUser(&auth.GetUserRequest{Name: walletAccount})
	if err != nil {
		return nil, fmt.Errorf("check user %s: %w", walletAccount, err)
	}

	for _, account := range policy.SupportAccounts {
		_, err := w.authClient.GetUser(&auth.GetUserRequest{Name: account})
		if err != nil {
			return nil, fmt.Errorf("check user %s: %w", account, err)
		}
	}

	go func() {
		channel := types.NewChannelInfo(ip, out)
		defer close(out)
		addrs, err := w.getValidatedAddress(ctx, channel, policy.SignBytes, walletAccount)
		if err != nil {
			log.Error(err)
			return
		}

		// register signer address to venus-auth
		err = w.registerSignerAddress(ctx, walletAccount, addrs...)
		if err != nil {
			log.Error(err)
			return
		}

		walletChannelInfo := newWalletChannelInfo(channel, addrs, policy.SignBytes)

		err = w.walletConnMgr.addNewConn(walletAccount, policy, walletChannelInfo)
		if err != nil {
			log.Errorf("validate address error %v", err)
			return
		}

		log.Infof("add new connections %s %s", walletAccount, walletChannelInfo.ChannelId)
		//todo rescan address to add new address or remove

		stats.Record(ctx, metrics.WalletRegister.M(1))
		stats.Record(ctx, metrics.WalletSource.M(1))

		connectBytes, err := json.Marshal(types2.ConnectedCompleted{
			ChannelId: walletChannelInfo.ChannelId,
		})
		if err != nil {
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
		stats.Record(ctx, metrics.WalletUnregister.M(1))
		if err = w.walletConnMgr.removeConn(walletAccount, walletChannelInfo); err != nil {
			log.Errorf("validate address error %v", err)
		}
	}()
	return out, nil
}

func (w *WalletEventStream) ResponseWalletEvent(ctx context.Context, resp *types2.ResponseEvent) error {
	return w.ResponseEvent(ctx, resp)
}

func (w *WalletEventStream) SupportNewAccount(ctx context.Context, channelId sharedTypes.UUID, account string) error {
	walletAccount, exit := jwtclient.CtxGetName(ctx)
	if !exit {
		return errors.New("unable to get account name in method SupportNewAccount request")
	}

	err := w.walletConnMgr.addSupportAccount(walletAccount, account)
	if err == nil {
		log.Infof("wallet %s add account %s", walletAccount, account)
	} else {
		log.Errorf("wallet %s add account %s failed %v", walletAccount, account, err)
	}

	return err
}

func (w *WalletEventStream) AddNewAddress(ctx context.Context, channelId sharedTypes.UUID, addrs []address.Address) error {
	walletAccount, exit := jwtclient.CtxGetName(ctx)
	if !exit {
		return errors.New("unable to get account name in method AddNewAddress request")
	}

	info, err := w.walletConnMgr.getConn(walletAccount, channelId)
	if err != nil {
		return err
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.WalletAccountKey, walletAccount))
	for _, addr := range addrs {
		if err := w.verifyAddress(ctx, addr, info.ChannelInfo, info.signBytes, walletAccount); err != nil {
			return err
		}
		_ = stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(metrics.WalletAddressKey, addr.String())},
			metrics.WalletAddAddr.M(1))
	}

	err = w.walletConnMgr.addNewAddress(walletAccount, channelId, addrs)
	if err != nil {
		log.Errorf("wallet %s add address %v failed %v", walletAccount, addrs, err)
		return err
	}
	log.Infof("wallet %s add address %v successful!", walletAccount, addrs)

	// register signer address to venus-auth
	return w.registerSignerAddress(ctx, walletAccount, addrs...)
}

func (w *WalletEventStream) RemoveAddress(ctx context.Context, channelId sharedTypes.UUID, addrs []address.Address) error {
	walletAccount, exit := jwtclient.CtxGetName(ctx)
	if !exit {
		return errors.New("unable to get account name in method RemoveAddress request")
	}
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.WalletAccountKey, walletAccount))
	for _, addr := range addrs {
		_ = stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(metrics.WalletAddressKey, addr.String())},
			metrics.WalletRemoveAddr.M(1))
	}
	err := w.walletConnMgr.removeAddress(walletAccount, channelId, addrs)
	if err == nil {
		log.Infof("wallet %s remove address %v", walletAccount, addrs)
	} else {
		log.Infof("wallet %s remove address %v failed %v", walletAccount, addrs, err)
	}

	return err
}

func (w *WalletEventStream) isSignerAddress(addr address.Address) bool {
	protocol := addr.Protocol()
	if protocol == address.SECP256K1 || protocol == address.BLS {
		return true
	}

	return false
}

func (w *WalletEventStream) getAccountOfSigner(addr address.Address) (string, error) {
	if !w.isSignerAddress(addr) {
		return "", fmt.Errorf("%s is not a signable address", addr.String())
	}

	user, err := w.authClient.GetUserBySigner(&auth.GetUserBySignerRequest{Signer: addr.String()})
	if err != nil {
		return "", err
	}

	return user.Name, nil
}

func (w *WalletEventStream) WalletHas(ctx context.Context, addr address.Address) (bool, error) {
	account, err := w.getAccountOfSigner(addr)
	if err != nil {
		return false, err
	}

	return w.walletConnMgr.hasWalletChannel(account, addr)
}

func (w *WalletEventStream) WalletSign(ctx context.Context, addr address.Address, toSign []byte, meta sharedTypes.MsgMeta) (*crypto.Signature, error) {
	payload, err := json.Marshal(&types2.WalletSignRequest{
		Signer: addr,
		ToSign: toSign,
		Meta:   meta,
	})
	if err != nil {
		return nil, err
	}

	account, err := w.getAccountOfSigner(addr)
	if err != nil {
		return nil, err
	}
	channels, err := w.walletConnMgr.getChannels(account, addr)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	var result crypto.Signature
	err = w.SendRequest(ctx, channels, "WalletSign", payload, &result)
	_ = stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(metrics.WalletAccountKey, account)},
		metrics.WalletSign.M(metrics.SinceInMilliseconds(start)))
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (w *WalletEventStream) ListWalletInfo(ctx context.Context) ([]*types2.WalletDetail, error) {
	return w.walletConnMgr.listWalletInfo(ctx)
}

func (w *WalletEventStream) ListWalletInfoByWallet(ctx context.Context, wallet string) (*types2.WalletDetail, error) {
	return w.walletConnMgr.listWalletInfoByWallet(ctx, wallet)
}

func (w *WalletEventStream) getValidatedAddress(ctx context.Context, channel *types.ChannelInfo, signBytes []byte, walletAccount string) ([]address.Address, error) {
	var addrs []address.Address

	start := time.Now()
	err := w.SendRequest(ctx, []*types.ChannelInfo{channel}, "WalletList", nil, &addrs)
	if err != nil {
		return nil, err
	}
	stats.Record(ctx, metrics.WalletList.M(metrics.SinceInMilliseconds(start)))

	// validate the wallet is really has the address
	validAddrs := make([]address.Address, 0, len(addrs))
	for _, addr := range addrs {
		if err := w.verifyAddress(ctx, addr, channel, signBytes, walletAccount); err != nil {
			return nil, err
		}
		validAddrs = append(validAddrs, addr)
		_ = stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(metrics.WalletAddressKey, addr.String())},
			metrics.WalletAddressNum.M(1))
	}

	return validAddrs, nil
}

func (w *WalletEventStream) verifyAddress(ctx context.Context, addr address.Address, channel *types.ChannelInfo, signBytes []byte, walletAccount string) error {
	if w.disableVerifyWalletAddrs {
		log.Infof("skip verify account:%s, address: %s, wallet address verification is disabled.",
			walletAccount, addr)
		return nil
	}
	signData := GetSignData(w.randBytes, signBytes)
	payload, err := json.Marshal(&types2.WalletSignRequest{
		Signer: addr,
		ToSign: signData,
		Meta:   sharedTypes.MsgMeta{Type: sharedTypes.MTVerifyAddress, Extra: w.randBytes},
	})
	if err != nil {
		return err
	}

	start := time.Now()
	var sig crypto.Signature
	err = w.SendRequest(ctx, []*types.ChannelInfo{channel}, "WalletSign", payload, &sig)
	stats.Record(ctx, metrics.WalletSign.M(metrics.SinceInMilliseconds(start)))
	if err != nil {
		return fmt.Errorf("wallet %s verify address %s failed, signed error %v", walletAccount, addr.String(), err)
	}
	if err := wcrypto.Verify(&sig, addr, signData); err != nil {
		return fmt.Errorf("wallet %s verify address %s failed: %v", walletAccount, addr.String(), err)
	}
	log.Infof("wallet %s verify address %s success", walletAccount, addr)

	return nil
}

func (w *WalletEventStream) registerSignerAddress(ctx context.Context, walletAccount string, addrs ...address.Address) error {
	for _, addr := range addrs {
		if !w.isSignerAddress(addr) {
			return fmt.Errorf("%s is not a signable address", addr.String())
		}

		bCreate, err := w.authClient.UpsertSigner(walletAccount, addr.String())
		if err != nil {
			return fmt.Errorf("upsert %s to venus-auth: %w", addr.String(), err)
		}

		opStr := "create"
		if !bCreate {
			opStr = "update"
		}
		log.Infof("venus-auth %s %s for user %s success.", opStr, addr.String(), walletAccount)
	}

	return nil
}

func GetSignData(datas ...[]byte) []byte {
	hasher := sha256.New()
	for _, data := range datas {
		_, _ = hasher.Write(data)
	}
	return hasher.Sum(nil)
}
