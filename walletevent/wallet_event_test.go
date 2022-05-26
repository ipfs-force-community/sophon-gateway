package walletevent

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/venus-auth/auth"
	"github.com/filecoin-project/venus-auth/cmd/jwtclient"
	vcrypto "github.com/filecoin-project/venus/pkg/crypto"
	_ "github.com/filecoin-project/venus/pkg/crypto/secp"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	"github.com/filecoin-project/venus/venus-shared/types/gateway"
	"github.com/ipfs-force-community/venus-gateway/types"
	"github.com/ipfs-force-community/venus-gateway/validator/mocks"
	"github.com/stretchr/testify/require"
)

func TestListenWalletEvent(t *testing.T) {

	t.Run("correct", func(t *testing.T) {
		walletAccount := "walletAccount" //nolint
		walletEvent := setupWalletEvent(t)
		//register
		policy := &gateway.WalletRegisterPolicy{
			SupportAccounts: []string{"admin"},
			SignBytes:       []byte{1, 2, 3},
		}

		ctx, cancel := context.WithCancel(context.Background())

		client := setupClient(t, walletAccount, policy, walletEvent)
		_ = client.listenWalletEvent(ctx, policy)
		go client.start(ctx)
		initBody := <-client.readyForInit
		walletInfo, err := walletEvent.ListWalletInfoByWallet(ctx, walletAccount)
		require.NoError(t, err)
		require.Equal(t, walletInfo.Account, walletAccount)
		require.Equal(t, walletInfo.SupportAccounts, []string{"admin"})
		require.Equal(t, walletInfo.ConnectStates[0].ChannelID, initBody.ChannelId)

		//cancel and got a close request channel
		cancel()
		client.waitClose()
	})

	t.Run("multiple listen", func(t *testing.T) {
		walletAccount := "walletAccount"
		walletEvent := setupWalletEvent(t)
		//register
		policy := &gateway.WalletRegisterPolicy{
			SupportAccounts: []string{"admin"},
			SignBytes:       []byte{1, 2, 3},
		}

		ctx, cancel := context.WithCancel(context.Background())

		client := setupClient(t, walletAccount, policy, walletEvent)
		_ = client.listenWalletEvent(ctx, policy)
		go client.start(ctx)
		<-client.readyForInit

		client2 := setupClient(t, walletAccount, policy, walletEvent)
		_ = client2.listenWalletEvent(ctx, policy)
		go client2.start(ctx)
		<-client2.readyForInit

		walletInfo, err := walletEvent.ListWalletInfoByWallet(ctx, walletAccount)
		require.NoError(t, err)
		require.Len(t, walletInfo.ConnectStates, 2)
		//cancel and got a close request channel
		cancel()
		client.waitClose()
		client2.waitClose()
	})

	t.Run("wallet account not found", func(t *testing.T) {
		walletAccount := "walletAccount"
		walletEvent := setupWalletEvent(t)
		//register
		policy := &gateway.WalletRegisterPolicy{
			SupportAccounts: []string{"admin"},
			SignBytes:       []byte{1, 2, 3},
		}

		ctx, cancel := context.WithCancel(context.Background())

		client := setupClient(t, walletAccount, policy, walletEvent)
		_, err := walletEvent.ListenWalletEvent(ctx, policy)
		go client.start(ctx)
		require.EqualError(t, err, "unable to get account name in method ListenWalletEvent request")

		//cancel and got a close request channel
		cancel()
	})
}

func TestSupportNewAccount(t *testing.T) {
	walletAccount := "walletAccount"
	//register
	policy := &gateway.WalletRegisterPolicy{
		SupportAccounts: []string{"admin"},
		SignBytes:       []byte{1, 2, 3},
	}
	ctx, cancel := context.WithCancel(context.Background())

	walletEvent := setupWalletEvent(t)
	client := setupClient(t, walletAccount, policy, walletEvent)
	_ = client.listenWalletEvent(ctx, policy)
	go client.start(ctx)
	<-client.readyForInit

	err := client.supportNewAccount(ctx, "ac1")
	require.NoError(t, err)

	wallets, err := walletEvent.ListWalletInfo(ctx)
	require.NoError(t, err)
	require.Len(t, wallets, 1)
	require.Len(t, wallets[0].SupportAccounts, 2)
	require.Contains(t, wallets[0].SupportAccounts, "ac1")
	require.Contains(t, wallets[0].SupportAccounts, "admin")

	err = walletEvent.SupportNewAccount(ctx, client.channelID, "fake_acc")
	require.EqualError(t, err, "unable to get account name in method SupportNewAccount request")

	ctx = jwtclient.CtxWithName(ctx, "fac_acc")
	err = walletEvent.SupportNewAccount(ctx, client.channelID, "__")
	require.NoError(t, err)
	//cancel and got a close request channel
	cancel()
	client.waitClose()
}

func TestAddNewAddress(t *testing.T) {
	walletAccount := "walletAccount"
	//register
	policy := &gateway.WalletRegisterPolicy{
		SupportAccounts: []string{"admin"},
		SignBytes:       []byte{1, 2, 3},
	}
	ctx, cancel := context.WithCancel(context.Background())

	walletEvent := setupWalletEvent(t)
	client := setupClient(t, walletAccount, policy, walletEvent)
	_ = client.listenWalletEvent(ctx, policy)
	go client.start(ctx)
	<-client.readyForInit

	addr1 := client.newkey()
	addr2 := client.newkey()

	err := walletEvent.AddNewAddress(ctx, client.channelID, []address.Address{addr1})
	require.EqualError(t, err, "unable to get account name in method AddNewAddress request")

	ctx = jwtclient.CtxWithName(ctx, walletAccount)
	err = walletEvent.AddNewAddress(ctx, client.channelID, []address.Address{addr1})
	require.NoError(t, err)

	ctx = jwtclient.CtxWithName(ctx, walletAccount)
	err = walletEvent.AddNewAddress(ctx, client.channelID, []address.Address{addr1}) //allow dup add
	require.NoError(t, err)

	ctx = jwtclient.CtxWithName(ctx, walletAccount)
	err = walletEvent.AddNewAddress(ctx, client.channelID, []address.Address{addr1, addr1, addr2})
	require.NoError(t, err)

	wallet, err := walletEvent.ListWalletInfoByWallet(ctx, walletAccount)
	require.NoError(t, err)
	require.Contains(t, wallet.ConnectStates[0].Addrs, addr2)
	require.Contains(t, wallet.ConnectStates[0].Addrs, addr1)
	//cancel and got a close request channel
	cancel()
	client.waitClose()
}

func TestRemoveNewAddressAndWalletHas(t *testing.T) {
	walletAccount := "walletAccount"
	//register
	policy := &gateway.WalletRegisterPolicy{
		SupportAccounts: []string{"admin"},
		SignBytes:       []byte{1, 2, 3},
	}
	ctx, cancel := context.WithCancel(context.Background())

	walletEvent := setupWalletEvent(t)
	client := setupClient(t, walletAccount, policy, walletEvent)
	_ = client.listenWalletEvent(ctx, policy)
	go client.start(ctx)
	<-client.readyForInit

	addr1 := client.newkey()
	accCtx := jwtclient.CtxWithName(ctx, walletAccount)
	err := walletEvent.AddNewAddress(accCtx, client.channelID, []address.Address{addr1})
	require.NoError(t, err)
	has, err := walletEvent.WalletHas(ctx, "admin", addr1)
	require.NoError(t, err)
	require.True(t, has)

	addr2 := client.newkey()
	err = walletEvent.AddNewAddress(accCtx, client.channelID, []address.Address{addr2})
	require.NoError(t, err)
	has, err = walletEvent.WalletHas(ctx, "admin", addr2)
	require.NoError(t, err)
	require.True(t, has)

	has, err = walletEvent.WalletHas(ctx, "fak_acc", addr1)
	require.NoError(t, err)
	require.False(t, has)

	err = walletEvent.RemoveAddress(ctx, client.channelID, []address.Address{addr1})
	require.EqualError(t, err, "unable to get account name in method RemoveAddress request")

	err = walletEvent.RemoveAddress(accCtx, client.channelID, []address.Address{addr1})
	require.NoError(t, err)

	has, err = walletEvent.WalletHas(ctx, "admin", addr1)
	require.NoError(t, err)
	require.False(t, has)

	wallet, err := walletEvent.ListWalletInfoByWallet(ctx, walletAccount)
	require.NoError(t, err)
	require.Len(t, wallet.ConnectStates[0].Addrs, 2)
	require.Contains(t, wallet.ConnectStates[0].Addrs, addr2)
	require.NotContains(t, wallet.ConnectStates[0].Addrs, addr1)

	err = walletEvent.RemoveAddress(accCtx, client.channelID, []address.Address{addr2})
	require.NoError(t, err)
	wallet, err = walletEvent.ListWalletInfoByWallet(ctx, walletAccount)
	require.NoError(t, err)
	require.Len(t, wallet.ConnectStates[0].Addrs, 1)

	has, err = walletEvent.WalletHas(ctx, "admin", addr1)
	require.NoError(t, err)
	require.False(t, has)
	//cancel and got a close request channel
	cancel()
	client.waitClose()
}

func TestWalletSign(t *testing.T) {
	walletAccount := "walletAccount"
	//register
	policy := &gateway.WalletRegisterPolicy{
		SupportAccounts: []string{},
		SignBytes:       []byte{1, 2, 3},
	}
	ctx, cancel := context.WithCancel(context.Background())

	walletEvent := setupWalletEvent(t)
	client := setupClient(t, walletAccount, policy, walletEvent)
	_ = client.listenWalletEvent(ctx, policy)
	go client.start(ctx)
	<-client.readyForInit

	addr, err := client.randAddr()
	require.NoError(t, err)
	_, err = walletEvent.WalletSign(ctx, "admin", addr, []byte{1, 2, 3}, sharedTypes.MsgMeta{
		Type:  sharedTypes.MTUnknown,
		Extra: nil,
	})
	require.Error(t, err)

	err = client.supportNewAccount(ctx, "admin")
	require.NoError(t, err)

	_, err = walletEvent.WalletSign(ctx, "admin", addr, []byte{1, 2, 3}, sharedTypes.MsgMeta{
		Type:  sharedTypes.MTUnknown,
		Extra: nil,
	})
	require.NoError(t, err)

	//cancel and got a close request channel
	cancel()
	client.waitClose()
}

func setupWalletEvent(t *testing.T) *WalletEventStream {
	authClient := mocks.NewMockAuthClient()
	authClient.AddMockUser(&auth.OutputUser{
		Id:         "id",
		Name:       "admin",
		SourceType: 0,
		Comment:    "",
		State:      0,
		CreateTime: 0,
		UpdateTime: 0,
	})

	ctx := context.Background()
	return NewWalletEventStream(ctx, authClient, types.DefaultConfig())
}

func setupClient(t *testing.T, walletAccount string, policy *gateway.WalletRegisterPolicy, event *WalletEventStream) *mockClient {
	pk1, err := vcrypto.NewSecpKeyFromSeed(rand.Reader)
	require.NoError(t, err)
	addr1, err := pk1.Address()
	require.NoError(t, err)

	return &mockClient{
		t:      t,
		policy: policy,
		keys: map[address.Address]vcrypto.KeyInfo{
			addr1: pk1,
		},
		readyForInit:  make(chan *gateway.ConnectedCompleted),
		event:         event,
		walletAccount: walletAccount,
	}
}

type mockClient struct {
	t             *testing.T
	channelID     sharedTypes.UUID
	walletAccount string
	policy        *gateway.WalletRegisterPolicy
	requestCh     chan *gateway.RequestEvent
	keys          map[address.Address]vcrypto.KeyInfo
	event         *WalletEventStream
	readyForInit  chan *gateway.ConnectedCompleted
}

func (m *mockClient) getkey(addr address.Address) (vcrypto.KeyInfo, error) {
	if key, ok := m.keys[addr]; ok {
		return key, nil
	}
	return vcrypto.KeyInfo{}, errors.New("not found key")
}

func (m *mockClient) randAddr() (address.Address, error) {
	for k := range m.keys {
		return k, nil
	}
	return address.Undef, errors.New("no key")
}
func (m *mockClient) waitClose() {
	select {
	case <-time.After(time.Second * 30):
		m.t.Errorf("unable to wait for closed channel within 30s")
	case _, ok := <-m.requestCh:
		if !ok {
			return
		}
	}
}

func (m *mockClient) newkey() address.Address {
	keyInfo, err := vcrypto.NewSecpKeyFromSeed(rand.Reader)
	require.NoError(m.t, err)
	addr1, err := keyInfo.Address()
	require.NoError(m.t, err)
	m.keys[addr1] = keyInfo
	return addr1
}

func (m *mockClient) listenWalletEvent(ctx context.Context, policy *gateway.WalletRegisterPolicy) chan *gateway.RequestEvent {
	ctx = jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1")
	ctx = jwtclient.CtxWithName(ctx, m.walletAccount)
	requestCh, err := m.event.ListenWalletEvent(ctx, policy)
	require.NoError(m.t, err)
	m.requestCh = requestCh
	return requestCh
}

func (m *mockClient) supportNewAccount(ctx context.Context, account string) error {
	ctx = jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1")
	ctx = jwtclient.CtxWithName(ctx, m.walletAccount)
	return m.event.SupportNewAccount(ctx, m.channelID, account)
}

func (m *mockClient) start(ctx context.Context) {
	//mock client
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-m.requestCh:
			{
				print(req.Method)
				if req.Method == "WalletSign" {
					var requestBody gateway.WalletSignRequest
					err := json.Unmarshal(req.Payload, &requestBody)
					require.NoError(m.t, err)
					keyInfo, err := m.getkey(requestBody.Signer)
					require.NoError(m.t, err)
					signData := GetSignData(requestBody.ToSign, m.policy.SignBytes)
					sig, err := vcrypto.Sign(signData, keyInfo.Key(), vcrypto.SigTypeSecp256k1)
					require.NoError(m.t, err)
					result, err := json.Marshal(sig)
					require.NoError(m.t, err)
					err = m.event.ResponseEvent(ctx, &gateway.ResponseEvent{
						ID:      req.ID,
						Payload: result,
						Error:   "",
					})
					require.NoError(m.t, err)
				} else if req.Method == "WalletList" {
					var addrs []address.Address
					for _, keyInfo := range m.keys {
						addr, err := keyInfo.Address()
						require.NoError(m.t, err)
						addrs = append(addrs, addr)
					}
					result, err := json.Marshal(addrs)
					require.NoError(m.t, err)
					err = m.event.ResponseEvent(ctx, &gateway.ResponseEvent{
						ID:      req.ID,
						Payload: result,
						Error:   "",
					})
					require.NoError(m.t, err)
				} else if req.Method == "InitConnect" {
					initBody := &gateway.ConnectedCompleted{}
					err := json.Unmarshal(req.Payload, initBody)
					require.NoError(m.t, err)
					m.channelID = initBody.ChannelId
					m.readyForInit <- initBody
				}
			}
		}
	}
}
