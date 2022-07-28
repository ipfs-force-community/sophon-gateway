package integrate

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-state-types/network"

	"github.com/filecoin-project/go-state-types/abi"

	"github.com/filecoin-project/venus/venus-shared/actors/builtin"

	"github.com/ipfs-force-community/metrics"
	"github.com/ipfs-force-community/venus-gateway/config"
	"github.com/ipfs-force-community/venus-gateway/testhelper"

	"github.com/ipfs-force-community/venus-gateway/proofevent"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/go-jsonrpc"

	"github.com/filecoin-project/venus/venus-shared/api"

	"github.com/filecoin-project/venus/venus-shared/api/gateway/v1"

	logging "github.com/ipfs/go-log/v2"

	"github.com/stretchr/testify/require"
)

func TestProofAPI(t *testing.T) {
	t.Run("compute proof", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mAddr, err := address.NewIDAddress(10)
		require.NoError(t, err)

		wsUrl, token := setupProofDaemon(t, []address.Address{mAddr}, ctx, defaultTestConfig())
		sAPi, sCloser, err := serverProofAPI(ctx, wsUrl, token)
		require.NoError(t, err)
		defer sCloser()

		walletEventClient, cCloser, err := proofevent.NewProofRegisterClient(ctx, wsUrl, token)
		require.NoError(t, err)
		defer cCloser()

		expectInfo := []builtin.ExtendedSectorInfo{
			{
				SealProof:    abi.RegisteredSealProof_StackedDrg2KiBV1_1,
				SectorNumber: 100,
				SectorKey:    nil,
				SealedCID:    cid.Undef,
			},
		}
		expectRand := []byte{1, 23}
		expectEpoch := abi.ChainEpoch(100)
		expectVersion := network.Version(10)
		expectProof := []builtin.PoStProof{
			{
				PoStProof:  abi.RegisteredPoStProof_StackedDrgWindow32GiBV1,
				ProofBytes: []byte{3, 4},
			},
		}
		handler := testhelper.NewProofHander(t, expectInfo, expectRand, expectEpoch, expectVersion, expectProof, false)
		proofEvent := proofevent.NewProofEvent(walletEventClient, mAddr, handler, logging.Logger("test").With())
		go proofEvent.ListenProofRequest(ctx)
		proofEvent.WaitReady(ctx)

		proof, err := sAPi.ComputeProof(ctx, mAddr, expectInfo, expectRand, expectEpoch, expectVersion)
		require.NoError(t, err)
		require.Equal(t, expectProof, proof)
	})

	t.Run("wait too long and all request timeout", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mAddr, err := address.NewIDAddress(10)
		require.NoError(t, err)

		wsUrl, token := setupProofDaemon(t, []address.Address{mAddr}, ctx, testConfig{
			requestTimeout: time.Second * 3,
			clearInterval:  time.Second * 3,
		})
		sAPi, sCloser, err := serverProofAPI(ctx, wsUrl, token)
		require.NoError(t, err)
		defer sCloser()

		walletEventClient, cCloser, err := proofevent.NewProofRegisterClient(ctx, wsUrl, token)
		require.NoError(t, err)
		defer cCloser()

		proofEvent := proofevent.NewProofEvent(walletEventClient, mAddr, &timeoutHandler{}, logging.Logger("test").With())
		go proofEvent.ListenProofRequest(ctx)
		proofEvent.WaitReady(ctx)

		var wg sync.WaitGroup
		var errs []error
		var lk sync.Mutex
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				_, err := sAPi.ComputeProof(ctx, mAddr, nil, nil, 0, 0)
				lk.Lock()
				defer lk.Unlock()
				errs = append(errs, err)
			}()
		}
		wg.Done()
		for _, err := range errs {
			require.Contains(t, err, "timer clean this request due to exceed wait time")
		}
	})

	t.Run("proof list connect", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		mAddr, err := address.NewIDAddress(10)
		require.NoError(t, err)
		maddr2, err := address.NewIDAddress(15)
		require.NoError(t, err)

		wsUrl, token := setupProofDaemon(t, []address.Address{mAddr, maddr2}, ctx, defaultTestConfig())
		sAPi, sCloser, err := serverProofAPI(ctx, wsUrl, token)
		require.NoError(t, err)
		defer sCloser()

		walletEventClient, cCloser, err := proofevent.NewProofRegisterClient(ctx, wsUrl, token)
		require.NoError(t, err)
		defer cCloser()

		expectInfo := []builtin.ExtendedSectorInfo{
			{
				SealProof:    abi.RegisteredSealProof_StackedDrg2KiBV1_1,
				SectorNumber: 100,
				SectorKey:    nil,
				SealedCID:    cid.Undef,
			},
		}
		expectRand := []byte{1, 23}
		expectEpoch := abi.ChainEpoch(100)
		expectVersion := network.Version(10)
		expectProof := []builtin.PoStProof{
			{
				PoStProof:  abi.RegisteredPoStProof_StackedDrgWindow32GiBV1,
				ProofBytes: []byte{3, 4},
			},
		}
		handler := testhelper.NewProofHander(t, expectInfo, expectRand, expectEpoch, expectVersion, expectProof, false)

		proofEvent := proofevent.NewProofEvent(walletEventClient, mAddr, handler, logging.Logger("test").With())
		go proofEvent.ListenProofRequest(ctx)
		proofEvent.WaitReady(ctx)

		proofEvent2 := proofevent.NewProofEvent(walletEventClient, mAddr, handler, logging.Logger("test").With())
		go proofEvent2.ListenProofRequest(ctx)
		proofEvent2.WaitReady(ctx)

		proofEvent3 := proofevent.NewProofEvent(walletEventClient, maddr2, handler, logging.Logger("test").With())
		go proofEvent3.ListenProofRequest(ctx)
		proofEvent3.WaitReady(ctx)

		miners, err := sAPi.ListConnectedMiners(ctx)
		require.NoError(t, err)
		require.Equal(t, 2, len(miners))

		minerConnections, err := sAPi.ListMinerConnection(ctx, mAddr)
		require.NoError(t, err)
		require.Equal(t, 2, len(minerConnections.Connections))
		require.Equal(t, 2, minerConnections.ConnectionCount)

		minerConnections2, err := sAPi.ListMinerConnection(ctx, maddr2)
		require.NoError(t, err)
		require.Equal(t, 1, len(minerConnections2.Connections))
		require.Equal(t, 1, minerConnections2.ConnectionCount)
	})
}

type timeoutHandler struct {
}

func (*timeoutHandler) ComputeProof(context.Context, []builtin.ExtendedSectorInfo, abi.PoStRandomness, abi.ChainEpoch, network.Version) ([]builtin.PoStProof, error) {
	time.Sleep(time.Hour)
	return nil, nil
}

func serverProofAPI(ctx context.Context, url, token string) (gateway.IProofEvent, jsonrpc.ClientCloser, error) {
	headers := http.Header{}
	headers.Add(api.AuthorizationHeader, "Bearer "+token)
	return gateway.NewIGatewayRPC(ctx, url, headers)
}

func setupProofDaemon(t *testing.T, validateMiner []address.Address, ctx context.Context, tCfg testConfig) (string, string) {
	cfg := &config.Config{
		API:       &config.APIConfig{ListenAddress: "/ip4/127.0.0.1/tcp/0"},
		Auth:      &config.AuthConfig{URL: "127.0.0.1:1"},
		Metrics:   config.DefaultConfig().Metrics,
		Trace:     &metrics.TraceConfig{JaegerTracingEnabled: false},
		RateLimit: &config.RateLimitCofnig{Redis: ""},
	}
	addr, token, err := MockMain(ctx, validateMiner, t.TempDir(), cfg, tCfg)
	require.NoError(t, err)
	url, err := url.Parse(addr)
	require.NoError(t, err)
	wsUrl := fmt.Sprintf("ws://127.0.0.1:%s/rpc/v1", url.Port())
	return wsUrl, string(token)
}
