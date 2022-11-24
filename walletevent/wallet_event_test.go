// stm: #unit
package walletevent

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-address"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/venus-auth/auth"
	"github.com/filecoin-project/venus-auth/jwtclient"
	_ "github.com/filecoin-project/venus/pkg/crypto/secp"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"

	"github.com/ipfs-force-community/venus-gateway/testhelper"
	"github.com/ipfs-force-community/venus-gateway/types"
	"github.com/ipfs-force-community/venus-gateway/validator/mocks"
)

func TestListenWalletEvent(t *testing.T) {
	walletAccount := "walletAccount"
	supportAccount := []string{"admin"}
	t.Run("correct", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		walletEvent := setupWalletEvent(t, walletAccount, supportAccount...)
		client := setupClient(t, ctx, walletAccount, supportAccount, walletEvent)
		{
			ctx := jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1")
			// stm: @VENUSGATEWAY_WALLET_EVENT_LISTEN_WALLET_EVENT_002
			err := client.walletEventClient.listenWalletRequestOnce(ctx)
			require.Error(t, err)
		}
		// stm: @VENUSGATEWAY_WALLET_EVENT_LISTEN_WALLET_EVENT_001, @VENUSGATEWAY_WALLET_EVENT_RESPONSE_WALLET_EVENT_001
		go client.listenWalletEvent(ctx)
		client.walletEventClient.WaitReady(ctx)

		walletInfo, err := walletEvent.ListWalletInfoByWallet(ctx, walletAccount)
		require.NoError(t, err)
		require.Equal(t, walletInfo.Account, walletAccount)
		require.Contains(t, walletInfo.SupportAccounts, "walletAccount")
		require.Contains(t, walletInfo.SupportAccounts, "admin")
	})

	t.Run("multiple listen", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		walletEvent := setupWalletEvent(t, walletAccount, supportAccount...)
		client := setupClient(t, ctx, walletAccount, []string{}, walletEvent)
		go client.listenWalletEvent(ctx)
		client.walletEventClient.WaitReady(ctx)

		client2 := setupClient(t, ctx, walletAccount, []string{}, walletEvent)
		go client2.listenWalletEvent(ctx)
		client2.walletEventClient.WaitReady(ctx)

		walletInfo, err := walletEvent.ListWalletInfoByWallet(ctx, walletAccount)
		require.NoError(t, err)
		require.Len(t, walletInfo.ConnectStates, 2)
	})

	t.Run("wallet account not found", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		walletEvent := setupWalletEvent(t, walletAccount, supportAccount...)
		// register
		client := setupClient(t, ctx, walletAccount, supportAccount, walletEvent)
		err := client.walletEventClient.listenWalletRequestOnce(ctx)
		require.Contains(t, err.Error(), "unable to get account name in method ListenWalletEvent request")
	})
}

func TestSupportNewAccount(t *testing.T) {
	walletAccount := "walletAccount"
	supportAccount := []string{"admin"}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	walletEvent := setupWalletEvent(t, walletAccount, supportAccount...)
	client := setupClient(t, ctx, walletAccount, supportAccount, walletEvent)
	go client.listenWalletEvent(ctx)
	client.walletEventClient.WaitReady(ctx)

	// stm: @VENUSGATEWAY_WALLET_EVENT_SUPPORT_NEW_ACCOUNT_001
	err := client.supportNewAccount(ctx, "ac1")
	require.NoError(t, err)

	wallets, err := walletEvent.ListWalletInfo(ctx)
	require.NoError(t, err)
	require.Len(t, wallets, 1)
	require.Len(t, wallets[0].SupportAccounts, 3)
	require.Contains(t, wallets[0].SupportAccounts, "ac1")
	require.Contains(t, wallets[0].SupportAccounts, "admin")
	require.Contains(t, wallets[0].SupportAccounts, "walletAccount")

	err = client.walletEventClient.SupportAccount(ctx, "fake_acc")
	require.EqualError(t, err, "unable to get account name in method SupportNewAccount request")

	ctx = jwtclient.CtxWithName(ctx, "fac_acc")
	err = client.walletEventClient.SupportAccount(ctx, "__")
	require.NoError(t, err)

	// wallet account not exists in context
	// stm: @VENUSGATEWAY_WALLET_EVENT_SUPPORT_NEW_ACCOUNT_002
	require.Error(t, client.walletEventClient.SupportAccount(context.Background(), "xxx_x"))
}

func TestAddNewAddress(t *testing.T) {
	walletAccount := "walletAccount"
	supportAccount := []string{"admin"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	walletEvent := setupWalletEvent(t, walletAccount, supportAccount...)
	client := setupClient(t, ctx, walletAccount, supportAccount, walletEvent)
	go client.listenWalletEvent(ctx)
	client.walletEventClient.WaitReady(ctx)

	addr1 := client.newkey()
	addr2 := client.newkey()

	// wallet account not exists in context
	// stm: @VENUSGATEWAY_WALLET_EVENT_ADD_NEW_ADDRESS_002
	err := client.walletEventClient.AddNewAddress(ctx, []address.Address{addr1})
	require.EqualError(t, err, "unable to get account name in method AddNewAddress request")

	ctx = jwtclient.CtxWithName(ctx, "incorrect wallet account")
	// wallet connection info not found
	// stm: @VENUSGATEWAY_WALLET_EVENT_ADD_NEW_ADDRESS_003
	err = client.walletEventClient.AddNewAddress(ctx, []address.Address{addr1})
	require.Error(t, err)

	ctx = jwtclient.CtxWithName(ctx, walletAccount)
	// stm: @VENUSGATEWAY_WALLET_EVENT_ADD_NEW_ADDRESS_004
	client.wallet.SetFail(ctx, true)
	walletEvent.disableVerifyWalletAddrs = false
	// verify address failed
	err = client.walletEventClient.AddNewAddress(ctx, []address.Address{addr1})
	require.Contains(t, err.Error(), "verify address")
	client.wallet.SetFail(ctx, false)
	walletEvent.disableVerifyWalletAddrs = true

	// stm: @VENUSGATEWAY_WALLET_EVENT_ADD_NEW_ADDRESS_001
	err = client.walletEventClient.AddNewAddress(ctx, []address.Address{addr1})
	require.NoError(t, err)

	ctx = jwtclient.CtxWithName(ctx, walletAccount)
	err = client.walletEventClient.AddNewAddress(ctx, []address.Address{addr1}) // allow dup add
	require.NoError(t, err)

	ctx = jwtclient.CtxWithName(ctx, walletAccount)
	err = client.walletEventClient.AddNewAddress(ctx, []address.Address{addr1, addr1, addr2})
	require.NoError(t, err)

	wallet, err := walletEvent.ListWalletInfoByWallet(ctx, walletAccount)
	require.NoError(t, err)
	require.Contains(t, wallet.ConnectStates[0].Addrs, addr2)
	require.Contains(t, wallet.ConnectStates[0].Addrs, addr1)
}

func TestRemoveNewAddressAndWalletHas(t *testing.T) {
	walletAccount := "walletAccount"
	supportAccount := []string{"admin"}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	walletEvent := setupWalletEvent(t, walletAccount, supportAccount...)
	// The supported account must contain the account corresponding to the API token
	client := setupClient(t, ctx, walletAccount, supportAccount, walletEvent)
	go client.listenWalletEvent(ctx)
	client.walletEventClient.WaitReady(ctx)

	addr1 := client.newkey()
	accCtx := jwtclient.CtxWithName(ctx, walletAccount)
	err := client.walletEventClient.AddNewAddress(accCtx, []address.Address{addr1})
	require.NoError(t, err)
	has, err := walletEvent.WalletHas(ctx, addr1, []string{walletAccount})
	require.NoError(t, err)
	require.True(t, has)

	addr2 := client.newkey()
	err = client.walletEventClient.AddNewAddress(accCtx, []address.Address{addr2})
	require.NoError(t, err)
	has, err = walletEvent.WalletHas(ctx, addr2, []string{walletAccount})
	require.NoError(t, err)
	require.True(t, has)

	has, err = walletEvent.WalletHas(ctx, addr1, []string{"fak_acc"})
	require.NoError(t, err)
	require.False(t, has)

	// stm: @VENUSGATEWAY_WALLET_EVENT_REMOVE_ADDRESS_002
	err = client.walletEventClient.RemoveAddress(ctx, []address.Address{addr1})
	require.EqualError(t, err, "unable to get account name in method RemoveAddress request")

	// stm: @VENUSGATEWAY_WALLET_EVENT_REMOVE_ADDRESS_001
	err = client.walletEventClient.RemoveAddress(accCtx, []address.Address{addr1})
	require.NoError(t, err)

	has, err = walletEvent.WalletHas(ctx, addr1, []string{walletAccount})
	require.NoError(t, err)
	require.False(t, has)

	wallet, err := walletEvent.ListWalletInfoByWallet(ctx, walletAccount)
	require.NoError(t, err)
	require.Len(t, wallet.ConnectStates[0].Addrs, 2)
	require.Contains(t, wallet.ConnectStates[0].Addrs, addr2)
	require.NotContains(t, wallet.ConnectStates[0].Addrs, addr1)

	err = client.walletEventClient.RemoveAddress(accCtx, []address.Address{addr2})
	require.NoError(t, err)
	wallet, err = walletEvent.ListWalletInfoByWallet(ctx, walletAccount)
	require.NoError(t, err)
	require.Len(t, wallet.ConnectStates[0].Addrs, 1)

	has, err = walletEvent.WalletHas(ctx, addr1, []string{walletAccount})
	require.NoError(t, err)
	require.False(t, has)
}

func TestWalletSign(t *testing.T) {
	walletAccount := "walletAccount"
	// register
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	walletEvent := setupWalletEvent(t, walletAccount)
	client := setupClient(t, ctx, walletAccount, []string{}, walletEvent)
	go client.listenWalletEvent(ctx)
	client.walletEventClient.WaitReady(ctx)

	// stm: @VENUSGATEWAY_WALLET_EVENT_WALLET_SIGN_003
	_, err := walletEvent.WalletSign(ctx, address.Undef, []string{"invalid account"}, []byte{1, 2, 3}, sharedTypes.MsgMeta{
		Type:  sharedTypes.MTUnknown,
		Extra: nil,
	})
	require.Error(t, err)

	addrs, err := client.wallet.WalletList(ctx)
	for _, addr := range addrs {
		require.NoError(t, err)
		// stm: @VENUSGATEWAY_WALLET_EVENT_WALLET_SIGN_001
		_, err = walletEvent.WalletSign(ctx, addr, []string{walletAccount}, []byte{1, 2, 3}, sharedTypes.MsgMeta{
			Type:  sharedTypes.MTUnknown,
			Extra: nil,
		})

		// Under the new mechanism, signer and walletAccount are bound to be bound, so it will be successful
		require.NoError(t, err)

		err = client.supportNewAccount(ctx, "admin")
		require.NoError(t, err)

		_, err = walletEvent.WalletSign(ctx, addr, []string{walletAccount}, []byte{1, 2, 3}, sharedTypes.MsgMeta{
			Type:  sharedTypes.MTUnknown,
			Extra: nil,
		})
		require.NoError(t, err)

		client.wallet.SetFail(ctx, true)
		_, err = walletEvent.WalletSign(ctx, addr, []string{walletAccount}, []byte{1, 2, 3}, sharedTypes.MsgMeta{
			Type:  sharedTypes.MTUnknown,
			Extra: nil,
		})
		require.EqualError(t, err, "mock error")
	}
}

func setupWalletEvent(t *testing.T, walletAccount string, accounts ...string) *WalletEventStream {
	users := make([]*auth.OutputUser, 0)
	for _, account := range accounts {
		users = append(users, &auth.OutputUser{
			Name: account,
		})
	}
	users = append(users, &auth.OutputUser{
		Name: walletAccount,
	})
	authClient := mocks.NewMockAuthClient()
	authClient.AddMockUser(users...)

	ctx := context.Background()
	return NewWalletEventStream(ctx, authClient, types.DefaultConfig(), true)
}

func setupClient(t *testing.T, ctx context.Context, walletAccount string, supportAccounts []string, event *WalletEventStream) *mockClient {
	wallet := testhelper.NewMemWallet()
	_, err := wallet.AddKey(context.Background())
	require.NoError(t, err)
	walletEventClient := NewWalletEventClient(ctx, wallet, event, logging.Logger("test").With(), supportAccounts)
	return &mockClient{
		t:                 t,
		walletEventClient: walletEventClient,
		wallet:            wallet,
		walletAccount:     walletAccount,
	}
}

type mockClient struct {
	t                 *testing.T
	wallet            *testhelper.MemWallet
	walletEventClient *WalletEventClient
	walletAccount     string
}

func (m *mockClient) newkey() address.Address {
	addr, err := m.wallet.AddKey(context.Background())
	require.NoError(m.t, err)
	return addr
}

func (m *mockClient) listenWalletEvent(ctx context.Context) {
	ctx = jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1")
	ctx = jwtclient.CtxWithName(ctx, m.walletAccount)
	m.walletEventClient.ListenWalletRequest(ctx)
}

func (m *mockClient) supportNewAccount(ctx context.Context, account string) error {
	ctx = jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1")
	ctx = jwtclient.CtxWithName(ctx, m.walletAccount)
	return m.walletEventClient.SupportAccount(ctx, account)
}

func TestGetSignBytes(t *testing.T) {
	for i := 0; i < 10; i++ {
		testGetSignBytes(t)
	}
}

func testGetSignBytes(t *testing.T) {
	getRandBytes := func() []byte {
		buf := make([]byte, 32)
		if _, err := rand.Read(buf); err != nil {
			panic(fmt.Sprintf("init random bytes for address verify failed:%s", err))
		}
		return buf
	}

	data1 := getRandBytes()
	data2 := getRandBytes()

	hasher := sha256.New()
	_, _ = hasher.Write(append(data1, data2...))
	signData1 := hasher.Sum(nil)

	signData2 := GetSignData(data1, data2)

	require.Equal(t, signData1, signData2)
}
