package integrate

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/gorilla/mux"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/plugin/ochttp"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"

	"github.com/ipfs-force-community/metrics"
	"github.com/ipfs-force-community/metrics/ratelimit"
	"github.com/ipfs-force-community/venus-gateway/config"

	"github.com/filecoin-project/venus-auth/auth"
	"github.com/filecoin-project/venus-auth/jwtclient"

	v2API "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"
	"github.com/filecoin-project/venus/venus-shared/api/permission"

	"github.com/ipfs-force-community/venus-gateway/api"
	"github.com/ipfs-force-community/venus-gateway/api/v1api"
	"github.com/ipfs-force-community/venus-gateway/marketevent"
	metrics2 "github.com/ipfs-force-community/venus-gateway/metrics"
	"github.com/ipfs-force-community/venus-gateway/proofevent"
	"github.com/ipfs-force-community/venus-gateway/types"
	"github.com/ipfs-force-community/venus-gateway/validator"
	"github.com/ipfs-force-community/venus-gateway/validator/mocks"
	"github.com/ipfs-force-community/venus-gateway/version"
	"github.com/ipfs-force-community/venus-gateway/walletevent"
)

var log = logging.Logger("mock main")

type testConfig struct {
	requestTimeout time.Duration
	clearInterval  time.Duration
}

func defaultTestConfig() testConfig {
	return testConfig{
		requestTimeout: time.Minute * 5,
		clearInterval:  time.Minute * 5,
	}
}

func MockMain(ctx context.Context, validateMiner []address.Address, repoPath string, cfg *config.Config, tcfg testConfig) (string, []byte, error) {
	requestCfg := &types.RequestConfig{
		RequestQueueSize: 30,
		RequestTimeout:   tcfg.requestTimeout,
		ClearInterval:    tcfg.clearInterval,
	}

	minerValidator := validator.MockAuthMinerValidator{ValidatedAddr: validateMiner}

	// In WalletEvent, IAuthClient must be called
	user := []*auth.OutputUser{
		{
			Name: "defaultLocalToken",
		},
		{
			Name: "admin",
		},
	}
	authClient := mocks.NewMockAuthClient()
	authClient.AddMockUser(ctx, user...)
	walletStream := walletevent.NewWalletEventStream(ctx, authClient, requestCfg, true)

	proofStream := proofevent.NewProofEventStream(ctx, minerValidator, requestCfg)
	marketStream := marketevent.NewMarketEventStream(ctx, minerValidator, &types.RequestConfig{
		RequestQueueSize: 30,
		RequestTimeout:   time.Hour * 12,
		ClearInterval:    time.Hour,
	})

	gatewayAPIImpl := api.NewGatewayAPIImpl(proofStream, walletStream, marketStream)

	log.Infof("venus-gateway current version %s", version.UserVersion)
	log.Info("Setting up control endpoint at " + cfg.API.ListenAddress)

	var fullNode v2API.IGatewayStruct
	permission.PermissionProxy(gatewayAPIImpl, &fullNode)
	gatewayAPI := (v2API.IGateway)(&fullNode)

	if len(cfg.RateLimit.Redis) > 0 {
		limiter, err := ratelimit.NewRateLimitHandler(cfg.RateLimit.Redis, nil,
			&jwtclient.ValueFromCtx{},
			authClient,
			logging.Logger("rate-limit"))
		_ = logging.SetLogLevel("rate-limit", "info")
		if err != nil {
			return "", nil, err
		}
		var rateLimitAPI v2API.IGatewayStruct
		limiter.ProxyLimitFullAPI(gatewayAPI, &rateLimitAPI)
		gatewayAPI = &rateLimitAPI
	}

	mux := mux.NewRouter()
	// v2api
	rpcServerV2 := jsonrpc.NewServer()
	rpcServerV2.Register("Gateway", gatewayAPI)
	mux.Handle("/rpc/v2", rpcServerV2)

	// v1api
	lowerFullNode := v1api.WrapperV2Full{IGateway: gatewayAPI}
	rpcServerV1 := jsonrpc.NewServer()
	rpcServerV1.Register("Gateway", lowerFullNode)
	mux.Handle("/rpc/v1", rpcServerV1)

	// v0api, once history
	// rpcServerV0 := jsonrpc.NewServer()
	// rpcServerV0.Register("Gateway", lowerFullNode)
	// mux.Handle("/rpc/v0", rpcServerV0)

	mux.PathPrefix("/").Handler(http.DefaultServeMux)

	localJwtCli, localToken, err := jwtclient.NewLocalAuthClient()
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate local jwt client: %v", err)
	}

	handler := (http.Handler)(jwtclient.NewAuthMux(localJwtCli, jwtclient.WarpIJwtAuthClient(authClient), mux))

	if err := metrics2.SetupMetrics(ctx, cfg.Metrics, gatewayAPIImpl); err != nil {
		return "", nil, err
	}

	log.Infof("trace config %v", cfg.Trace)
	repoter, err := metrics.RegisterJaeger(cfg.Trace.ServerName, cfg.Trace)
	if err != nil {
		return "", nil, fmt.Errorf("register jaeger exporter failed %v", cfg.Trace)
	}
	if repoter != nil {
		log.Info("register jaeger exporter success!")

		defer metrics.UnregisterJaeger(repoter)
		handler = &ochttp.Handler{Handler: handler}
	}

	srv := httptest.NewServer(handler)
	return srv.URL, localToken, nil
}
