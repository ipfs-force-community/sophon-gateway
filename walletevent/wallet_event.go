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

	"github.com/filecoin-project/venus-auth/jwtclient"

	wcrypto "github.com/filecoin-project/venus/pkg/crypto"
	v2API "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	sharedGatewayTypes "github.com/filecoin-project/venus/venus-shared/types/gateway"

	"github.com/ipfs-force-community/venus-gateway/metrics"
	"github.com/ipfs-force-community/venus-gateway/types"
)

var log = logging.Logger("event_stream")

var _ v2API.IWalletClient = (*WalletEventStream)(nil)

type WalletEventStream struct {
	walletConnMgr IWalletConnMgr
	cfg           *types.RequestConfig
	authClient    jwtclient.IAuthClient
	randBytes     []byte
	*types.BaseEventStream

	disableVerifyWalletAddrs bool
}

func NewWalletEventStream(ctx context.Context, authClient jwtclient.IAuthClient, cfg *types.RequestConfig, diableVerifyWalletAddrs bool) *WalletEventStream {
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

func (w *WalletEventStream) ListenWalletEvent(ctx context.Context, policy *sharedGatewayTypes.WalletRegisterPolicy) (<-chan *sharedGatewayTypes.RequestEvent, error) {
	walletAccount, exit := jwtclient.CtxGetName(ctx)
	if !exit {
		return nil, errors.New("unable to get account name in method ListenWalletEvent request")
	}

	ip, _ := jwtclient.CtxGetTokenLocation(ctx) // todo sure exit?
	out := make(chan *sharedGatewayTypes.RequestEvent, w.cfg.RequestQueueSize)
	walletLog := log.With("account", walletAccount).With("ip", ip)
	ctx, _ = tag.New(ctx, tag.Upsert(metrics.WalletAccountKey, walletAccount), tag.Upsert(metrics.IPKey, ip))

	// Verify account: must exist in venus-auth
	accounts := policy.SupportAccounts
	accounts = append(accounts, walletAccount)
	err := w.authClient.VerifyUsers(ctx, accounts)
	if err != nil {
		return nil, fmt.Errorf("verify user %v failed: %w", accounts, err)
	}

	go func() {
		channel := types.NewChannelInfo(ctx, ip, out)
		defer close(out)
		addrs, err := w.getValidatedAddress(ctx, channel, policy.SignBytes, walletAccount)
		if err != nil {
			walletLog.Errorf("unable to value address %v", err)
			return
		}

		walletChannelInfo := newWalletChannelInfo(channel, addrs, policy.SignBytes)
		err = w.walletConnMgr.addNewConn(walletAccount, policy, walletChannelInfo)
		if err != nil {
			walletLog.Errorf("validate address error %v", err)
			return
		}
		walletLog.Infof("add new connections %s", walletChannelInfo.ChannelId)

		// register signer address to venus-auth
		for _, account := range accounts {
			if err := w.registerSignerAddress(ctx, account, addrs...); err != nil {
				log.Errorf("register %v for %s failed: %v", addrs, account, err)
				continue
			}
			log.Infof("register %v for %s success", addrs, account)
		}

		// todo rescan address to add new address or remove

		stats.Record(ctx, metrics.WalletRegister.M(1))

		connectBytes, err := json.Marshal(sharedGatewayTypes.ConnectedCompleted{
			ChannelId: walletChannelInfo.ChannelId,
		})
		if err != nil {
			walletLog.Errorf("marshal failed %v", err)
			return
		}

		out <- &sharedGatewayTypes.RequestEvent{
			ID:         sharedTypes.NewUUID(),
			Method:     "InitConnect",
			CreateTime: time.Now(),
			Payload:    connectBytes,
			Result:     nil,
		} // not response

		<-ctx.Done()
		stats.Record(ctx, metrics.WalletUnregister.M(1))
		if err = w.walletConnMgr.removeConn(walletAccount, walletChannelInfo); err != nil {
			walletLog.Errorf("remove connect error %v", err)
		} else { // nolint
			// The records bound to the system will not have a lot of records, and there will be no additional effects.
			// There are expenses and other potential risks for each disconnection of betting sales.
			// Therefore, it is not used first.
			//// unregister all signer of this account
			//signers := make([]address.Address, len(walletChannelInfo.addrs))
			//idx := 0
			//for addr := range walletChannelInfo.addrs {
			//	signers[idx] = addr
			//	idx++
			//}
			//
			//if err := w.unregisterSignerAddress(ctx, walletAccount, signers...); err != nil {
			//	log.Errorf("unregister %v for %s failed: %w", signers, walletAccount, err)
			//} else {
			//	log.Infof("unregister %v for %s success", signers, walletAccount)
			//}
		}
	}()
	return out, nil
}

func (w *WalletEventStream) ResponseWalletEvent(ctx context.Context, resp *sharedGatewayTypes.ResponseEvent) error {
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
	walletDetail, err := w.walletConnMgr.listWalletInfoByWallet(ctx, walletAccount)
	if err != nil {
		log.Errorf("get wallet %s info failed %v", walletAccount, err)
		return err
	}
	for _, account := range walletDetail.SupportAccounts {
		if err := w.registerSignerAddress(ctx, account, addrs...); err != nil {
			log.Errorf("register %v for %s failed: %v", addrs, account, err)
			continue
		}
		log.Infof("register %v for %s success", addrs, account)
	}

	return nil
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
	if err != nil {
		log.Infof("wallet %s remove address %v failed %v", walletAccount, addrs, err)
		return err
	}

	log.Infof("wallet %s remove address %v", walletAccount, addrs)

	// The records bound to the system will not have a lot of records, and there will be no additional effects.
	// There are expenses and other potential risks for each disconnection of betting sales.
	// Therefore, it is not used first.
	//// unregister signer address to venus-auth
	//if err := w.unregisterSignerAddress(ctx, walletAccount, addrs...); err != nil {
	//	log.Errorf("unregister %v for %s failed: %w", addrs, walletAccount, err)
	//	return err
	//}
	//log.Infof("unregister %v for %s success", addrs, walletAccount)

	return nil
}

func (w *WalletEventStream) isSignerAddress(addr address.Address) bool {
	protocol := addr.Protocol()
	if protocol == address.SECP256K1 || protocol == address.BLS {
		return true
	}

	return false
}

func (w *WalletEventStream) WalletHas(ctx context.Context, addr address.Address, accounts []string) (bool, error) {
	for _, account := range accounts {
		bHas, err := w.walletConnMgr.hasWalletChannel(account, addr)
		if err != nil {
			return false, err
		}

		if bHas {
			return true, nil
		}
	}

	return false, nil
}

func (w *WalletEventStream) WalletSign(ctx context.Context, addr address.Address, accounts []string, toSign []byte, meta sharedTypes.MsgMeta) (*crypto.Signature, error) {
	channels := make([]*types.ChannelInfo, 0)
	for _, account := range accounts {
		cs, err := w.walletConnMgr.getChannels(account, addr)
		if err != nil {
			// It should continue as long as the channel can be found
			log.Warnf("get channel for %s of %s: %s", account, addr, err.Error())
			continue
		}

		channels = append(channels, cs...)
	}

	start := time.Now()
	var result crypto.Signature
	payload, err := json.Marshal(&sharedGatewayTypes.WalletSignRequest{
		Signer: addr,
		ToSign: toSign,
		Meta:   meta,
	})
	if err != nil {
		return nil, err
	}
	err = w.SendRequest(ctx, channels, "WalletSign", payload, &result)
	_ = stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(metrics.WalletAccountKey, fmt.Sprintf("%v", accounts))},
		metrics.WalletSign.M(metrics.SinceInMilliseconds(start)))
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (w *WalletEventStream) ListWalletInfo(ctx context.Context) ([]*sharedGatewayTypes.WalletDetail, error) {
	return w.walletConnMgr.listWalletInfo(ctx)
}

func (w *WalletEventStream) ListWalletInfoByWallet(ctx context.Context, wallet string) (*sharedGatewayTypes.WalletDetail, error) {
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
	payload, err := json.Marshal(&sharedGatewayTypes.WalletSignRequest{
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
	signers := make([]address.Address, 0, len(addrs))
	for _, addr := range addrs {
		if !w.isSignerAddress(addr) {
			log.Warnf("%s is not a signable address", addr.String())
			continue
		}

		signers = append(signers, addr)
	}

	return w.authClient.RegisterSigners(ctx, walletAccount, signers)
}

// nolint: unused
func (w *WalletEventStream) unregisterSignerAddress(ctx context.Context, walletAccount string, addrs ...address.Address) error {
	signers := make([]address.Address, 0, len(addrs))
	for _, addr := range addrs {
		if !w.isSignerAddress(addr) {
			log.Warnf("%s is not a signable address", addr.String())
			continue
		}

		signers = append(signers, addr)
	}

	return w.authClient.UnregisterSigners(ctx, walletAccount, signers)
}

func GetSignData(datas ...[]byte) []byte {
	hasher := sha256.New()
	for _, data := range datas {
		_, _ = hasher.Write(data)
	}
	return hasher.Sum(nil)
}
