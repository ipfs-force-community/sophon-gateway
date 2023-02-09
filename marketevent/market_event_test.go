// stm: #unit
package marketevent

import (
	"context"
	"testing"
	"time"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/venus-auth/auth"
	"github.com/filecoin-project/venus-auth/jwtclient"

	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	mktypes "github.com/filecoin-project/venus/venus-shared/types/market"

	"github.com/ipfs-force-community/venus-gateway/testhelper"
	"github.com/ipfs-force-community/venus-gateway/types"
	"github.com/ipfs-force-community/venus-gateway/validator"
	"github.com/ipfs-force-community/venus-gateway/validator/mocks"

	"github.com/stretchr/testify/require"
)

func TestListenMarketEvent(t *testing.T) {
	supportAccount := "client_account"
	t.Run("listen market request", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		minerAddr := address.NewForTestGetter()()
		// register
		marketEvent := setupMarketEvent(t, supportAccount, minerAddr)

		client := NewMarketEventClient(marketEvent, minerAddr, nil, log.With())
		go client.ListenMarketRequest(jwtclient.CtxWithName(jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1"), supportAccount))
		client.WaitReady(ctx)
	})

	t.Run("miner validate fail", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		addrGetter := address.NewForTestGetter()
		minerAddr := addrGetter()

		marketEvent := setupMarketEvent(t, supportAccount, minerAddr)
		client := NewMarketEventClient(marketEvent, addrGetter(), nil, log.With())
		// stm: @VENUSGATEWAY_MARKET_EVENT_LISTEN_MARKET_EVENT_002
		err := client.listenMarketRequestOnce(jwtclient.CtxWithName(jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1"), supportAccount))
		require.Contains(t, err.Error(), "not exist")
	})

	t.Run("ip not exit", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		addrGetter := address.NewForTestGetter()
		minerAddr := addrGetter()
		// register
		marketEvent := setupMarketEvent(t, supportAccount, minerAddr)
		client := NewMarketEventClient(marketEvent, minerAddr, nil, log.With())
		// stm: @VENUSGATEWAY_MARKET_EVENT_LISTEN_MARKET_EVENT_003
		err := client.listenMarketRequestOnce(jwtclient.CtxWithName(ctx, supportAccount))
		require.Contains(t, err.Error(), "ip not exist")
	})
}

func TestIsUnsealed(t *testing.T) {
	supportAccount := "client_account"
	addrGetter := address.NewForTestGetter()
	minerAddr := addrGetter()

	t.Run("correct", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		marketEvent := setupMarketEvent(t, supportAccount, minerAddr)
		handler := testhelper.NewMarketHandler(t)
		client := NewMarketEventClient(marketEvent, minerAddr, handler, log.With())
		// stm: @VENUSGATEWAY_MARKET_EVENT_RESPONSE_MARKET_EVENT_001
		go client.ListenMarketRequest(jwtclient.CtxWithName(jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1"), supportAccount))
		client.WaitReady(ctx)

		sid := abi.SectorNumber(10)
		size := abi.PaddedPieceSize(100)
		offset := sharedTypes.PaddedByteIndex(100)
		handler.SetCheckIsUnsealExpect(minerAddr, sid, offset, size, false)
		// stm: @VENUSGATEWAY_MARKET_EVENT_IS_UNSEALED_001
		isUnsealed, err := marketEvent.IsUnsealed(ctx, minerAddr, cid.Undef, sid, offset, size)
		require.NoError(t, err)
		require.True(t, isUnsealed)
	})

	t.Run("miner not found", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		marketEvent := setupMarketEvent(t, supportAccount, minerAddr)
		handler := testhelper.NewMarketHandler(t)
		client := NewMarketEventClient(marketEvent, minerAddr, handler, log.With())
		// stm: @VENUSGATEWAY_MARKET_EVENT_RESPONSE_MARKET_EVENT_002
		go client.ListenMarketRequest(jwtclient.CtxWithName(jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1"), supportAccount))
		client.WaitReady(ctx)

		sid := abi.SectorNumber(10)
		size := abi.PaddedPieceSize(100)
		offset := sharedTypes.PaddedByteIndex(100)
		handler.SetCheckIsUnsealExpect(minerAddr, sid, offset, size, false)
		// stm: @VENUSGATEWAY_MARKET_EVENT_IS_UNSEALED_002
		_, err := marketEvent.IsUnsealed(ctx, addrGetter(), cid.Undef, sid, offset, size)
		require.Contains(t, err.Error(), "no connections for this miner")
	})

	t.Run("response error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		marketEvent := setupMarketEvent(t, supportAccount, minerAddr)
		handler := testhelper.NewMarketHandler(t)
		client := NewMarketEventClient(marketEvent, minerAddr, handler, log.With())
		// stm: @VENUSGATEWAY_MARKET_EVENT_RESPONSE_MARKET_EVENT_003
		go client.ListenMarketRequest(jwtclient.CtxWithName(jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1"), supportAccount))
		client.WaitReady(ctx)

		sid := abi.SectorNumber(10)
		size := abi.PaddedPieceSize(100)
		offset := sharedTypes.PaddedByteIndex(100)
		handler.SetCheckIsUnsealExpect(minerAddr, sid, offset, size, true)
		// stm: @VENUSGATEWAY_MARKET_EVENT_IS_UNSEALED_003
		_, err := marketEvent.IsUnsealed(ctx, minerAddr, cid.Undef, sid, offset, size)
		require.EqualError(t, err, "mock error")
	})
}

func TestUnsealed(t *testing.T) {
	walletAccount := "client_account"
	addrGetter := address.NewForTestGetter()
	minerAddr := addrGetter()

	t.Run("correct", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		marketEvent := setupMarketEvent(t, walletAccount, minerAddr)
		handler := testhelper.NewMarketHandler(t)
		client := NewMarketEventClient(marketEvent, minerAddr, handler, log.With())
		go client.ListenMarketRequest(jwtclient.CtxWithName(jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1"), walletAccount))
		client.WaitReady(ctx)

		sid := abi.SectorNumber(10)
		size := abi.PaddedPieceSize(100)
		offset := sharedTypes.PaddedByteIndex(100)
		transfer := mktypes.Transfer{}
		pieceCid, err := cid.Decode("bafy2bzaced2kktxdkqw5pey5of3wtahz5imm7ta4ymegah466dsc5fonj73u2")
		require.NoError(t, err)
		handler.SetSectorsUnsealPieceExpect(pieceCid, minerAddr, sid, offset, size, transfer, false)
		// stm: @VENUSGATEWAY_MARKET_EVENT_SECTORS_UNSEAL_PIECE_001
		err = marketEvent.SectorsUnsealPiece(ctx, minerAddr, pieceCid, sid, offset, size, &transfer)
		require.NoError(t, err)
	})

	t.Run("miner not found", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		marketEvent := setupMarketEvent(t, walletAccount, minerAddr)
		handler := testhelper.NewMarketHandler(t)
		client := NewMarketEventClient(marketEvent, minerAddr, handler, log.With())
		go client.ListenMarketRequest(jwtclient.CtxWithName(jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1"), walletAccount))
		client.WaitReady(ctx)

		sid := abi.SectorNumber(10)
		size := abi.PaddedPieceSize(100)
		offset := sharedTypes.PaddedByteIndex(100)
		transfer := mktypes.Transfer{}
		pieceCid, err := cid.Decode("bafy2bzaced2kktxdkqw5pey5of3wtahz5imm7ta4ymegah466dsc5fonj73u2")
		require.NoError(t, err)
		handler.SetSectorsUnsealPieceExpect(pieceCid, minerAddr, sid, offset, size, transfer, false)
		// stm: @VENUSGATEWAY_MARKET_EVENT_SECTORS_UNSEAL_PIECE_002
		err = marketEvent.SectorsUnsealPiece(ctx, addrGetter(), pieceCid, sid, offset, size, &transfer)
		require.Contains(t, err.Error(), "no connections for this miner")
	})

	t.Run("response error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		marketEvent := setupMarketEvent(t, walletAccount, minerAddr)
		handler := testhelper.NewMarketHandler(t)
		client := NewMarketEventClient(marketEvent, minerAddr, handler, log.With())
		go client.ListenMarketRequest(jwtclient.CtxWithName(jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1"), walletAccount))
		client.WaitReady(ctx)

		sid := abi.SectorNumber(10)
		size := abi.PaddedPieceSize(100)
		offset := sharedTypes.PaddedByteIndex(100)
		transfer := mktypes.Transfer{}
		pieceCid, err := cid.Decode("bafy2bzaced2kktxdkqw5pey5of3wtahz5imm7ta4ymegah466dsc5fonj73u2")
		require.NoError(t, err)
		handler.SetSectorsUnsealPieceExpect(pieceCid, minerAddr, sid, offset, size, transfer, true)
		// stm: @VENUSGATEWAY_MARKET_EVENT_SECTORS_UNSEAL_PIECE_003
		err = marketEvent.SectorsUnsealPiece(ctx, minerAddr, pieceCid, sid, offset, size, &transfer)
		require.EqualError(t, err, "mock error")
	})
}

func TestListMarketConnectionsState(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	walletAccount := "client_account"
	minerAddr := address.NewForTestGetter()()
	// register
	marketEvent := setupMarketEvent(t, walletAccount, minerAddr)
	handler := testhelper.NewMarketHandler(t)
	client := NewMarketEventClient(marketEvent, minerAddr, handler, log.With())
	go client.ListenMarketRequest(jwtclient.CtxWithName(jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1"), walletAccount))
	client.WaitReady(ctx)

	// stm: @VENUSGATEWAY_MARKET_EVENT_LIST_MARKET_CONNECTIONS_STATE_001
	marketState, err := marketEvent.ListMarketConnectionsState(ctx)
	require.NoError(t, err)
	require.Len(t, marketState, 1)
	require.Equal(t, marketState[0].Addr, minerAddr)
}

func setupMarketEvent(t *testing.T, userName string, miners ...address.Address) *MarketEventStream {
	ctx := context.Background()
	authClient := mocks.NewMockAuthClient()
	user := &auth.OutputUser{
		Id:         "id",
		Name:       userName,
		Comment:    "",
		State:      1,
		CreateTime: 0,
		UpdateTime: 0,
		Miners:     []*auth.OutputMiner{},
	}
	for _, m := range miners {
		user.Miners = append(user.Miners, &auth.OutputMiner{
			Miner:     m,
			User:      userName,
			CreatedAt: time.Time{},
			UpdatedAt: time.Time{},
		},
		)
	}
	authClient.AddMockUser(ctx, user)

	return NewMarketEventStream(ctx, validator.NewMinerValidator(authClient), types.DefaultConfig())
}
