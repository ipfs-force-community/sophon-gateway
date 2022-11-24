// stm: #integration
package integrate

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/venus/venus-shared/api"
	v2API "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"
	types2 "github.com/filecoin-project/venus/venus-shared/types"

	"github.com/ipfs-force-community/metrics"

	"github.com/ipfs-force-community/venus-gateway/config"
	"github.com/ipfs-force-community/venus-gateway/testhelper"
	"github.com/ipfs-force-community/venus-gateway/walletevent"

	"github.com/stretchr/testify/require"
)

func TestWalletAPI(t *testing.T) {
	t.Run("wallet support account", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		wsUrl, token := setupDaemon(t, ctx)
		sAPi, sCloser, err := serverWalletAPI(ctx, wsUrl, token)
		require.NoError(t, err)
		defer sCloser()

		walletEventClient, cCloser, err := walletevent.NewWalletRegisterClient(ctx, wsUrl, token)
		require.NoError(t, err)
		defer cCloser()

		wallet := testhelper.NewMemWallet()
		_, err = wallet.AddKey(ctx)
		require.NoError(t, err)
		_, err = wallet.AddKey(ctx)
		require.NoError(t, err)

		walletEvent := walletevent.NewWalletEventClient(ctx, wallet, walletEventClient, logging.Logger("test").With(), []string{"admin"})
		go walletEvent.ListenWalletRequest(ctx)
		walletEvent.WaitReady(ctx)
		err = walletEvent.SupportAccount(ctx, "123")
		require.NoError(t, err)

		walletDetail, err := sAPi.ListWalletInfoByWallet(ctx, "defaultLocalToken")
		require.NoError(t, err)
		require.Contains(t, walletDetail.SupportAccounts, "123")
	})

	t.Run("wallet add new address", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		wsUrl, token := setupDaemon(t, ctx)
		sAPi, sCloser, err := serverWalletAPI(ctx, wsUrl, token)
		require.NoError(t, err)
		defer sCloser()

		walletEventClient, cCloser, err := walletevent.NewWalletRegisterClient(ctx, wsUrl, token)
		require.NoError(t, err)
		defer cCloser()

		wallet := testhelper.NewMemWallet()
		walletEvent := walletevent.NewWalletEventClient(ctx, wallet, walletEventClient, logging.Logger("test").With(), []string{"admin"})
		go walletEvent.ListenWalletRequest(ctx)
		walletEvent.WaitReady(ctx)

		toAddAddr1, err := wallet.AddKey(ctx)
		require.NoError(t, err)
		toAddAddr2, err := wallet.AddKey(ctx)
		require.NoError(t, err)
		err = walletEvent.AddNewAddress(ctx, []address.Address{toAddAddr1, toAddAddr2})
		require.NoError(t, err)

		// stm: @VENUSGATEWAY_API_LIST_WALLET_INFO_BY_WALLET_001
		walletDetail, err := sAPi.ListWalletInfoByWallet(ctx, "defaultLocalToken")
		require.NoError(t, err)
		require.Contains(t, walletDetail.ConnectStates[0].Addrs, toAddAddr1)
		require.Contains(t, walletDetail.ConnectStates[0].Addrs, toAddAddr2)
	})

	t.Run("wallet remove address", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		wsUrl, token := setupDaemon(t, ctx)
		sAPi, sCloser, err := serverWalletAPI(ctx, wsUrl, token)
		require.NoError(t, err)
		defer sCloser()

		walletEventClient, cCloser, err := walletevent.NewWalletRegisterClient(ctx, wsUrl, token)
		require.NoError(t, err)
		defer cCloser()

		wallet := testhelper.NewMemWallet()
		toRemoveAddr1, err := wallet.AddKey(ctx)
		require.NoError(t, err)
		toRemoveAddr2, err := wallet.AddKey(ctx)
		require.NoError(t, err)

		walletEvent := walletevent.NewWalletEventClient(ctx, wallet, walletEventClient, logging.Logger("test").With(), []string{"admin"})
		go walletEvent.ListenWalletRequest(ctx)
		walletEvent.WaitReady(ctx)

		err = walletEvent.RemoveAddress(ctx, []address.Address{toRemoveAddr1})
		require.NoError(t, err)
		walletDetail, err := sAPi.ListWalletInfoByWallet(ctx, "defaultLocalToken")
		require.NoError(t, err)
		require.NotContains(t, walletDetail.ConnectStates[0].Addrs, toRemoveAddr1)
		require.Contains(t, walletDetail.ConnectStates[0].Addrs, toRemoveAddr2)

		err = walletEvent.RemoveAddress(ctx, []address.Address{toRemoveAddr2})
		require.NoError(t, err)

		walletDetail, err = sAPi.ListWalletInfoByWallet(ctx, "defaultLocalToken")
		require.NoError(t, err)
		require.NotContains(t, walletDetail.ConnectStates[0].Addrs, toRemoveAddr2)
	})

	t.Run("wallet remove multiple address", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		wsUrl, token := setupDaemon(t, ctx)
		sAPi, sCloser, err := serverWalletAPI(ctx, wsUrl, token)
		require.NoError(t, err)
		defer sCloser()

		walletEventClient, cCloser, err := walletevent.NewWalletRegisterClient(ctx, wsUrl, token)
		require.NoError(t, err)
		defer cCloser()

		wallet := testhelper.NewMemWallet()
		toRemoveAddr1, err := wallet.AddKey(ctx)
		require.NoError(t, err)
		toRemoveAddr2, err := wallet.AddKey(ctx)
		require.NoError(t, err)

		walletEvent := walletevent.NewWalletEventClient(ctx, wallet, walletEventClient, logging.Logger("test").With(), []string{"admin"})
		go walletEvent.ListenWalletRequest(ctx)
		walletEvent.WaitReady(ctx)

		err = walletEvent.RemoveAddress(ctx, []address.Address{toRemoveAddr1, toRemoveAddr2})
		require.NoError(t, err)
		walletDetail, err := sAPi.ListWalletInfoByWallet(ctx, "defaultLocalToken")
		require.NoError(t, err)
		require.NotContains(t, walletDetail.ConnectStates[0].Addrs, toRemoveAddr1)
		require.NotContains(t, walletDetail.ConnectStates[0].Addrs, toRemoveAddr2)
	})

	t.Run("wallet list wallet info", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		wsUrl, token := setupDaemon(t, ctx)
		sAPi, sCloser, err := serverWalletAPI(ctx, wsUrl, token)
		require.NoError(t, err)
		defer sCloser()

		walletEventClient, cCloser, err := walletevent.NewWalletRegisterClient(ctx, wsUrl, token)
		require.NoError(t, err)
		defer cCloser()

		ctxClient01, cancelClient01 := context.WithCancel(context.Background())
		defer cancelClient01()
		wallet := testhelper.NewMemWallet()
		addr1, err := wallet.AddKey(ctxClient01)
		require.NoError(t, err)

		walletEvent := walletevent.NewWalletEventClient(ctxClient01, wallet, walletEventClient, logging.Logger("test").With(), []string{"admin"})
		go walletEvent.ListenWalletRequest(ctxClient01)
		walletEvent.WaitReady(ctxClient01)

		ctxClient02, cancelClient02 := context.WithCancel(context.Background())
		defer cancelClient02()
		wallet2 := testhelper.NewMemWallet()
		addr2, err := wallet2.AddKey(ctxClient02)
		require.NoError(t, err)

		walletEvent2 := walletevent.NewWalletEventClient(ctxClient02, wallet2, walletEventClient, logging.Logger("test").With(), []string{"admin"})
		go walletEvent2.ListenWalletRequest(ctxClient02)
		walletEvent2.WaitReady(ctxClient02)

		// stm: @VENUSGATEWAY_API_WALLET_SIGN_001
		walletInfo, err := sAPi.ListWalletInfo(ctx)
		require.NoError(t, err)
		require.Len(t, walletInfo, 1)
		require.Len(t, walletInfo[0].ConnectStates, 2)
		require.Len(t, walletInfo[0].ConnectStates[0].Addrs, 1)
		require.Len(t, walletInfo[0].ConnectStates[1].Addrs, 1)
		addrs := []address.Address{addr1, addr2}
		require.Contains(t, addrs, walletInfo[0].ConnectStates[1].Addrs[0])
		require.Contains(t, addrs, walletInfo[0].ConnectStates[0].Addrs[0])
	})

	t.Run("wallet wallet has", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		wsUrl, token := setupDaemon(t, ctx)
		sAPi, sCloser, err := serverWalletAPI(ctx, wsUrl, token)
		require.NoError(t, err)
		defer sCloser()

		walletEventClient, cCloser, err := walletevent.NewWalletRegisterClient(ctx, wsUrl, token)
		require.NoError(t, err)
		defer cCloser()

		wallet := testhelper.NewMemWallet()
		addr1, err := wallet.AddKey(ctx)
		require.NoError(t, err)

		wallet2 := testhelper.NewMemWallet()
		addr2, err := wallet2.AddKey(ctx)
		require.NoError(t, err)

		walletEvent := walletevent.NewWalletEventClient(ctx, wallet, walletEventClient, logging.Logger("test").With(), []string{"admin"})
		go walletEvent.ListenWalletRequest(ctx)
		walletEvent.WaitReady(ctx)

		// Including `walletEventClient.supportAccounts` and token-account of venus-wallet
		accounts := []string{"defaultLocalToken", "admin"}
		// stm: @VENUSGATEWAY_API_WALLET_HAS_001
		has, err := sAPi.WalletHas(ctx, addr1, accounts)
		require.NoError(t, err)
		require.True(t, has)

		has, err = sAPi.WalletHas(ctx, addr2, accounts)
		require.NoError(t, err)
		require.False(t, has)
	})

	t.Run("wallet wallet sign and verify", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		wsUrl, token := setupDaemon(t, ctx)
		sAPi, sCloser, err := serverWalletAPI(ctx, wsUrl, token)
		require.NoError(t, err)
		defer sCloser()

		walletEventClient, cCloser, err := walletevent.NewWalletRegisterClient(ctx, wsUrl, token)
		require.NoError(t, err)
		defer cCloser()

		wallet := testhelper.NewMemWallet()
		addr1, err := wallet.AddKey(ctx)
		require.NoError(t, err)

		walletEvent := walletevent.NewWalletEventClient(ctx, wallet, walletEventClient, logging.Logger("test").With(), []string{"admin"})
		go walletEvent.ListenWalletRequest(ctx)
		walletEvent.WaitReady(ctx)
		err = walletEvent.SupportAccount(ctx, "newAccount")
		require.NoError(t, err)

		for i := 0; i < 5; i++ {
			var msg [32]byte
			_, err = rand.Read(msg[:])
			require.NoError(t, err)
			// todo 模拟 账户存在
			// stm: @VENUSGATEWAY_API_WALLET_SIGN_001
			sig, err := sAPi.WalletSign(ctx, addr1, []string{"newAccount"}, msg[:], types2.MsgMeta{})
			require.NoError(t, err)
			err = wallet.Verify(ctx, addr1, sig, msg[:])
			require.NoError(t, err)
		}
	})
}

func serverWalletAPI(ctx context.Context, url, token string) (v2API.IWalletEvent, jsonrpc.ClientCloser, error) {
	headers := http.Header{}
	headers.Add(api.AuthorizationHeader, "Bearer "+token)
	return v2API.NewIGatewayRPC(ctx, url, headers)
}

func setupDaemon(t *testing.T, ctx context.Context) (string, string) {
	cfg := &config.Config{
		API:       &config.APIConfig{ListenAddress: "/ip4/127.0.0.1/tcp/0"},
		Auth:      &config.AuthConfig{URL: "127.0.0.1:1"}, // nouse
		Metrics:   config.DefaultConfig().Metrics,
		Trace:     &metrics.TraceConfig{JaegerTracingEnabled: false},
		RateLimit: &config.RateLimitCofnig{Redis: ""},
	}

	addr, token, err := MockMain(ctx, nil, t.TempDir(), cfg, defaultTestConfig())
	require.NoError(t, err)
	url, err := url.Parse(addr)
	require.NoError(t, err)
	wsUrl := fmt.Sprintf("ws://127.0.0.1:%s/rpc/v2", url.Port())
	return wsUrl, string(token)
}
