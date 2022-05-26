package marketevent

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"
	"github.com/filecoin-project/venus-auth/auth"
	"github.com/filecoin-project/venus-auth/cmd/jwtclient"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	"github.com/filecoin-project/venus/venus-shared/types/gateway"
	"github.com/ipfs-force-community/venus-gateway/types"
	"github.com/ipfs-force-community/venus-gateway/validator"
	"github.com/ipfs-force-community/venus-gateway/validator/mocks"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
)

func TestListenMarketEvent(t *testing.T) {
	t.Run("correct", func(t *testing.T) {
		walletAccount := "client_account" //nolint
		minerAddr := address.NewForTestGetter()()
		//register
		policy := &gateway.MarketRegisterPolicy{
			Miner: minerAddr,
		}
		ctx, cancel := context.WithCancel(context.Background())

		marketEvent := setupMarketEvent(t, "client_account", minerAddr)
		client := setupClient(t, walletAccount, policy, marketEvent)
		_ = client.listenWalletEvent(ctx, policy)
		go client.start(ctx)
		<-client.readyForInit

		//cancel and got a close request channel
		cancel()
		client.waitClose()
	})

	t.Run("miner validate fail", func(t *testing.T) {
		walletAccount := "client_account"
		addrGetter := address.NewForTestGetter()
		minerAddr := addrGetter()
		//register
		policy := &gateway.MarketRegisterPolicy{
			Miner: minerAddr,
		}
		ctx, cancel := context.WithCancel(context.Background())

		marketEvent := setupMarketEvent(t, "client_account", addrGetter())
		client := setupClient(t, walletAccount, policy, marketEvent)
		go client.start(ctx)
		defer cancel()

		ctx = jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1")
		ctx = jwtclient.CtxWithName(ctx, walletAccount)
		_, err := marketEvent.ListenMarketEvent(ctx, policy)
		require.Contains(t, err.Error(), "not exists")
	})

	t.Run("ip not exit", func(t *testing.T) {
		walletAccount := "client_account"
		addrGetter := address.NewForTestGetter()
		minerAddr := addrGetter()
		//register
		policy := &gateway.MarketRegisterPolicy{
			Miner: minerAddr,
		}
		ctx, cancel := context.WithCancel(context.Background())

		marketEvent := setupMarketEvent(t, "client_account", minerAddr)
		client := setupClient(t, walletAccount, policy, marketEvent)
		go client.start(ctx)
		defer cancel()
		_, err := marketEvent.ListenMarketEvent(ctx, policy)
		require.EqualError(t, err, "ip not exist")
	})
}

func TestIsUnsealed(t *testing.T) {
	walletAccount := "client_account"
	addrGetter := address.NewForTestGetter()
	minerAddr := addrGetter()
	//register
	policy := &gateway.MarketRegisterPolicy{
		Miner: minerAddr,
	}

	t.Run("correct", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		marketEvent := setupMarketEvent(t, "client_account", minerAddr)
		client := setupClient(t, walletAccount, policy, marketEvent)

		_ = client.listenWalletEvent(ctx, policy)
		go client.start(ctx)
		<-client.readyForInit

		isUnsealed, err := client.isUnsealed(ctx, minerAddr, cid.Undef, storage.SectorRef{
			ID: abi.SectorID{
				Miner:  abi.ActorID(5),
				Number: 10,
			},
			ProofType: abi.RegisteredSealProof_StackedDrg2KiBV1_1,
		}, sharedTypes.PaddedByteIndex(100), abi.PaddedPieceSize(100))
		require.NoError(t, err)
		require.True(t, isUnsealed)

		//cancel and got a close request channel
		cancel()
		client.waitClose()
	})

	t.Run("miner not found", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		marketEvent := setupMarketEvent(t, "client_account", minerAddr)
		client := setupClient(t, walletAccount, policy, marketEvent)

		_ = client.listenWalletEvent(ctx, policy)
		go client.start(ctx)
		<-client.readyForInit

		_, err := client.isUnsealed(ctx, addrGetter(), cid.Undef, storage.SectorRef{
			ID: abi.SectorID{
				Miner:  abi.ActorID(5),
				Number: 10,
			},
			ProofType: abi.RegisteredSealProof_StackedDrg2KiBV1_1,
		}, sharedTypes.PaddedByteIndex(100), abi.PaddedPieceSize(100))
		require.Contains(t, err.Error(), "no connections for this miner")

		//cancel and got a close request channel
		cancel()
		client.waitClose()
	})

	t.Run("response error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		marketEvent := setupMarketEvent(t, "client_account", minerAddr)
		client := setupClient(t, walletAccount, policy, marketEvent)

		_ = client.listenWalletEvent(ctx, policy)
		go client.start(ctx)
		<-client.readyForInit

		client.failSectorNumber = 20
		_, err := client.isUnsealed(ctx, minerAddr, cid.Undef, storage.SectorRef{
			ID: abi.SectorID{
				Miner:  abi.ActorID(5),
				Number: 20,
			},
			ProofType: abi.RegisteredSealProof_StackedDrg2KiBV1_1,
		}, sharedTypes.PaddedByteIndex(100), abi.PaddedPieceSize(100))
		require.EqualError(t, err, "mock error")

		//cancel and got a close request channel
		cancel()
		client.waitClose()
	})
}

func TestUnsealed(t *testing.T) {
	walletAccount := "client_account"
	addrGetter := address.NewForTestGetter()
	minerAddr := addrGetter()
	//register
	policy := &gateway.MarketRegisterPolicy{
		Miner: minerAddr,
	}

	t.Run("correct", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		marketEvent := setupMarketEvent(t, "client_account", minerAddr)
		client := setupClient(t, walletAccount, policy, marketEvent)

		_ = client.listenWalletEvent(ctx, policy)
		go client.start(ctx)
		<-client.readyForInit

		err := client.sectorsUnsealPiece(ctx, minerAddr, cid.Undef, storage.SectorRef{
			ID: abi.SectorID{
				Miner:  abi.ActorID(5),
				Number: 10,
			},
			ProofType: abi.RegisteredSealProof_StackedDrg2KiBV1_1,
		}, sharedTypes.PaddedByteIndex(100), abi.PaddedPieceSize(100), "mock path")
		require.NoError(t, err)

		//cancel and got a close request channel
		cancel()
		client.waitClose()
	})

	t.Run("miner not found", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		marketEvent := setupMarketEvent(t, "client_account", minerAddr)
		client := setupClient(t, walletAccount, policy, marketEvent)

		_ = client.listenWalletEvent(ctx, policy)
		go client.start(ctx)
		<-client.readyForInit

		err := client.sectorsUnsealPiece(ctx, addrGetter(), cid.Undef, storage.SectorRef{
			ID: abi.SectorID{
				Miner:  abi.ActorID(5),
				Number: 10,
			},
			ProofType: abi.RegisteredSealProof_StackedDrg2KiBV1_1,
		}, sharedTypes.PaddedByteIndex(100), abi.PaddedPieceSize(100), "mock path")
		require.Contains(t, err.Error(), "no connections for this miner")

		//cancel and got a close request channel
		cancel()
		client.waitClose()
	})

	t.Run("response error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		marketEvent := setupMarketEvent(t, "client_account", minerAddr)
		client := setupClient(t, walletAccount, policy, marketEvent)

		_ = client.listenWalletEvent(ctx, policy)
		go client.start(ctx)
		<-client.readyForInit

		client.failSectorNumber = 20
		err := client.sectorsUnsealPiece(ctx, minerAddr, cid.Undef, storage.SectorRef{
			ID: abi.SectorID{
				Miner:  abi.ActorID(5),
				Number: 20,
			},
			ProofType: abi.RegisteredSealProof_StackedDrg2KiBV1_1,
		}, sharedTypes.PaddedByteIndex(100), abi.PaddedPieceSize(100), "mock path")
		require.EqualError(t, err, "mock error")

		//cancel and got a close request channel
		cancel()
		client.waitClose()
	})
}

func TestListMarketConnectionsState(t *testing.T) {
	walletAccount := "client_account"
	minerAddr := address.NewForTestGetter()()
	//register
	policy := &gateway.MarketRegisterPolicy{
		Miner: minerAddr,
	}
	ctx, cancel := context.WithCancel(context.Background())

	marketEvent := setupMarketEvent(t, "client_account", minerAddr)
	client := setupClient(t, walletAccount, policy, marketEvent)
	_ = client.listenWalletEvent(ctx, policy)
	go client.start(ctx)
	<-client.readyForInit

	marketState, err := marketEvent.ListMarketConnectionsState(ctx)
	require.NoError(t, err)
	require.Len(t, marketState, 1)

	require.Equal(t, marketState[0].Addr, minerAddr)
	//cancel and got a close request channel
	cancel()
	client.waitClose()
}

func setupMarketEvent(t *testing.T, userName string, miners ...address.Address) *MarketEventStream {
	authClient := mocks.NewMockAuthClient()
	user := &auth.OutputUser{
		Id:         "id",
		Name:       userName,
		SourceType: 0,
		Comment:    "",
		State:      1,
		CreateTime: 0,
		UpdateTime: 0,
		Miners:     []*auth.OutputMiner{},
	}
	for _, m := range miners {
		user.Miners = append(user.Miners, &auth.OutputMiner{
			Miner:     m.String(),
			User:      userName,
			CreatedAt: time.Time{},
			UpdatedAt: time.Time{},
		},
		)
	}
	authClient.AddMockUser(user)
	ctx := context.Background()
	return NewMarketEventStream(ctx, validator.NewMinerValidator(authClient), types.DefaultConfig())
}

func setupClient(t *testing.T, account string, policy *gateway.MarketRegisterPolicy, event *MarketEventStream) *mockClient {
	return &mockClient{
		t:            t,
		account:      account,
		policy:       policy,
		readyForInit: make(chan *gateway.ConnectedCompleted),
		event:        event,
	}
}

type mockClient struct {
	t            *testing.T
	account      string
	channelID    sharedTypes.UUID
	policy       *gateway.MarketRegisterPolicy
	requestCh    chan *gateway.RequestEvent
	event        *MarketEventStream
	readyForInit chan *gateway.ConnectedCompleted

	expectPieceCid   cid.Cid
	expectSectorRef  storage.SectorRef
	expectOffset     sharedTypes.PaddedByteIndex
	expectSize       abi.PaddedPieceSize
	expectDest       string
	failSectorNumber abi.SectorNumber
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

func (m *mockClient) isUnsealed(ctx context.Context, miner address.Address, pieceCid cid.Cid, sector storage.SectorRef, offset sharedTypes.PaddedByteIndex, size abi.PaddedPieceSize) (bool, error) {
	m.expectPieceCid = pieceCid
	m.expectSectorRef = sector
	m.expectOffset = offset
	m.expectSize = size
	return m.event.IsUnsealed(ctx, miner, pieceCid, sector, offset, size)
}

func (m *mockClient) sectorsUnsealPiece(ctx context.Context, miner address.Address, pieceCid cid.Cid, sector storage.SectorRef, offset sharedTypes.PaddedByteIndex, size abi.PaddedPieceSize, dest string) error {
	m.expectPieceCid = pieceCid
	m.expectSectorRef = sector
	m.expectOffset = offset
	m.expectSize = size
	m.expectDest = dest
	return m.event.SectorsUnsealPiece(ctx, miner, pieceCid, sector, offset, size, dest)
}

func (m *mockClient) listenWalletEvent(ctx context.Context, policy *gateway.MarketRegisterPolicy) chan *gateway.RequestEvent {
	ctx = jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1")
	ctx = jwtclient.CtxWithName(ctx, m.account)
	requestCh, err := m.event.ListenMarketEvent(ctx, policy)
	require.NoError(m.t, err)
	m.requestCh = requestCh
	return requestCh
}

func (m *mockClient) start(ctx context.Context) {
	//mock client
	for {
		select {
		case <-ctx.Done():
			return
		case req := <-m.requestCh:
			{
				fmt.Println(req.Method)
				if req.Method == "IsUnsealed" {
					var unsealReq gateway.IsUnsealRequest
					err := json.Unmarshal(req.Payload, &unsealReq)
					require.NoError(m.t, err)
					if m.failSectorNumber == unsealReq.Sector.ID.Number {
						err = m.event.ResponseEvent(ctx, &gateway.ResponseEvent{
							ID:      req.ID,
							Payload: nil,
							Error:   "mock error",
						})
						require.NoError(m.t, err)
						continue
					}
					require.Equal(m.t, m.expectPieceCid, unsealReq.PieceCid)
					require.Equal(m.t, m.expectSectorRef, unsealReq.Sector)
					require.Equal(m.t, m.expectOffset, unsealReq.Offset)
					require.Equal(m.t, m.expectSize, unsealReq.Size)
					result, err := json.Marshal(true)
					require.NoError(m.t, err)
					err = m.event.ResponseEvent(ctx, &gateway.ResponseEvent{
						ID:      req.ID,
						Payload: result,
						Error:   "",
					})
					require.NoError(m.t, err)
				} else if req.Method == "SectorsUnsealPiece" {
					var unsealReq gateway.UnsealRequest
					err := json.Unmarshal(req.Payload, &unsealReq)
					require.NoError(m.t, err)
					if m.failSectorNumber == unsealReq.Sector.ID.Number {
						err = m.event.ResponseEvent(ctx, &gateway.ResponseEvent{
							ID:      req.ID,
							Payload: nil,
							Error:   "mock error",
						})
						require.NoError(m.t, err)
						continue
					}

					require.Equal(m.t, m.expectPieceCid, unsealReq.PieceCid)
					require.Equal(m.t, m.expectSectorRef, unsealReq.Sector)
					require.Equal(m.t, m.expectOffset, unsealReq.Offset)
					require.Equal(m.t, m.expectSize, unsealReq.Size)
					require.Equal(m.t, m.expectDest, unsealReq.Dest)
					err = m.event.ResponseEvent(ctx, &gateway.ResponseEvent{
						ID:      req.ID,
						Payload: nil,
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
