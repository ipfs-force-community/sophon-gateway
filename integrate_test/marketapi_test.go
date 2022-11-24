// stm: #integration
package integrate

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/venus/venus-shared/api"
	v2API "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"

	"github.com/ipfs-force-community/metrics"

	"github.com/ipfs-force-community/venus-gateway/config"
	"github.com/ipfs-force-community/venus-gateway/marketevent"
	"github.com/ipfs-force-community/venus-gateway/testhelper"
)

func TestMarketAPI(t *testing.T) {
	t.Run("check is unseal", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mAddr, err := address.NewIDAddress(10)
		require.NoError(t, err)

		wsUrl, token := setupMarketDaemon(t, []address.Address{mAddr}, ctx)
		sAPi, sCloser, err := serverMarketAPI(ctx, wsUrl, token)
		require.NoError(t, err)
		defer sCloser()

		walletEventClient, cCloser, err := marketevent.NewMarketRegisterClient(ctx, wsUrl, token)
		require.NoError(t, err)
		defer cCloser()

		handler := testhelper.NewMarketHandler(t)
		proofEvent := marketevent.NewMarketEventClient(walletEventClient, mAddr, handler, logging.Logger("test").With())
		// stm: @VENUSGATEWAY_MARKET_EVENT_LISTEN_MARKET_EVENT_001
		go proofEvent.ListenMarketRequest(ctx)
		proofEvent.WaitReady(ctx)

		sectorRef := storage.SectorRef{
			ID: abi.SectorID{
				Miner:  abi.ActorID(5),
				Number: 10,
			},
			ProofType: abi.RegisteredSealProof_StackedDrg2KiBV1_1,
		}
		size := abi.PaddedPieceSize(100)
		offset := sharedTypes.PaddedByteIndex(100)
		handler.SetCheckIsUnsealExpect(sectorRef, offset, size, false)
		// stm: @VENUSGATEWAY_API_IS_UNSEALED_001, @VENUSGATEWAY_MARKET_EVENT_IS_UNSEALED_001
		isUnseal, err := sAPi.IsUnsealed(ctx, mAddr, cid.Undef, sectorRef, offset, size)
		require.NoError(t, err)
		require.True(t, isUnseal)

		handler.SetCheckIsUnsealExpect(sectorRef, offset, size, true)
		_, err = sAPi.IsUnsealed(ctx, mAddr, cid.Undef, sectorRef, offset, size)
		require.EqualError(t, err, "mock error")
	})

	t.Run("unseal api", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mAddr, err := address.NewIDAddress(10)
		require.NoError(t, err)

		wsUrl, token := setupMarketDaemon(t, []address.Address{mAddr}, ctx)
		sAPi, sCloser, err := serverMarketAPI(ctx, wsUrl, token)
		require.NoError(t, err)
		defer sCloser()

		walletEventClient, cCloser, err := marketevent.NewMarketRegisterClient(ctx, wsUrl, token)
		require.NoError(t, err)
		defer cCloser()

		handler := testhelper.NewMarketHandler(t)
		proofEvent := marketevent.NewMarketEventClient(walletEventClient, mAddr, handler, logging.Logger("test").With())
		go proofEvent.ListenMarketRequest(ctx)
		proofEvent.WaitReady(ctx)

		sectorRef := storage.SectorRef{
			ID: abi.SectorID{
				Miner:  abi.ActorID(5),
				Number: 10,
			},
			ProofType: abi.RegisteredSealProof_StackedDrg2KiBV1_1,
		}
		size := abi.PaddedPieceSize(100)
		offset := sharedTypes.PaddedByteIndex(100)
		dest := "mock dest path"
		pieceCid, err := cid.Decode("bafy2bzaced2kktxdkqw5pey5of3wtahz5imm7ta4ymegah466dsc5fonj73u2")
		require.NoError(t, err)
		handler.SetSectorsUnsealPieceExpect(pieceCid, sectorRef, offset, size, dest, false)
		// stm: @VENUSGATEWAY_API_SECTOR_UNSEAL_PRICE_001
		err = sAPi.SectorsUnsealPiece(ctx, mAddr, pieceCid, sectorRef, offset, size, dest)
		require.NoError(t, err)

		handler.SetSectorsUnsealPieceExpect(pieceCid, sectorRef, offset, size, dest, true)
		err = sAPi.SectorsUnsealPiece(ctx, mAddr, pieceCid, sectorRef, offset, size, dest)
		require.EqualError(t, err, "mock error")
	})

	t.Run("unseal api", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mAddr, err := address.NewIDAddress(10)
		require.NoError(t, err)
		mAddr2, err := address.NewIDAddress(12)
		require.NoError(t, err)

		wsUrl, token := setupMarketDaemon(t, []address.Address{mAddr, mAddr2}, ctx)
		sAPi, sCloser, err := serverMarketAPI(ctx, wsUrl, token)
		require.NoError(t, err)
		defer sCloser()

		walletEventClient, cCloser, err := marketevent.NewMarketRegisterClient(ctx, wsUrl, token)
		require.NoError(t, err)
		defer cCloser()

		handler := testhelper.NewMarketHandler(t)
		marketEventClient := marketevent.NewMarketEventClient(walletEventClient, mAddr, handler, logging.Logger("test").With())
		go marketEventClient.ListenMarketRequest(ctx)
		marketEventClient.WaitReady(ctx)

		// stm: @VENUSGATEWAY_API_LIST_MARKET_CONNECTIONS_STATE_001
		connectsState, err := sAPi.ListMarketConnectionsState(ctx)
		require.NoError(t, err)
		require.Len(t, connectsState, 1)
		require.Equal(t, mAddr, connectsState[0].Addr)

		// add another
		marketEventClient2 := marketevent.NewMarketEventClient(walletEventClient, mAddr2, handler, logging.Logger("test").With())
		go marketEventClient2.ListenMarketRequest(ctx)
		marketEventClient2.WaitReady(ctx)

		connectsState, err = sAPi.ListMarketConnectionsState(ctx)
		require.NoError(t, err)
		require.Len(t, connectsState, 2)
		require.Contains(t, []address.Address{mAddr, mAddr2}, connectsState[0].Addr)
		require.Contains(t, []address.Address{mAddr, mAddr2}, connectsState[1].Addr)
	})
}

func serverMarketAPI(ctx context.Context, url, token string) (v2API.IMarketClient, jsonrpc.ClientCloser, error) {
	headers := http.Header{}
	headers.Add(api.AuthorizationHeader, "Bearer "+token)
	return v2API.NewIGatewayRPC(ctx, url, headers)
}

func setupMarketDaemon(t *testing.T, validateMiner []address.Address, ctx context.Context) (string, string) {
	cfg := &config.Config{
		API:       &config.APIConfig{ListenAddress: "/ip4/127.0.0.1/tcp/0"},
		Auth:      &config.AuthConfig{URL: "127.0.0.1:1"},
		Metrics:   config.DefaultConfig().Metrics,
		Trace:     &metrics.TraceConfig{JaegerTracingEnabled: false},
		RateLimit: &config.RateLimitCofnig{Redis: ""},
	}

	addr, token, err := MockMain(ctx, validateMiner, t.TempDir(), cfg, defaultTestConfig())
	require.NoError(t, err)
	url, err := url.Parse(addr)
	require.NoError(t, err)
	wsUrl := fmt.Sprintf("ws://127.0.0.1:%s/rpc/v2", url.Port())
	return wsUrl, string(token)
}
