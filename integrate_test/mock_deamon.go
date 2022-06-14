package integrate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/filecoin-project/go-address"

	"github.com/ipfs-force-community/venus-gateway/utils"

	"github.com/filecoin-project/go-jsonrpc"

	"github.com/filecoin-project/venus-auth/cmd/jwtclient"
	sharedGatewayApiV1 "github.com/filecoin-project/venus/venus-shared/api/gateway/v1"
	"github.com/filecoin-project/venus/venus-shared/api/permission"
	"github.com/gorilla/mux"
	"github.com/ipfs-force-community/metrics"
	"github.com/ipfs-force-community/metrics/ratelimit"
	"github.com/ipfs-force-community/venus-gateway/api"
	"github.com/ipfs-force-community/venus-gateway/marketevent"
	"github.com/ipfs-force-community/venus-gateway/proofevent"
	"github.com/ipfs-force-community/venus-gateway/types"
	"github.com/ipfs-force-community/venus-gateway/validator"
	"github.com/ipfs-force-community/venus-gateway/version"
	"github.com/ipfs-force-community/venus-gateway/walletevent"
	logging "github.com/ipfs/go-log/v2"
	"go.opencensus.io/plugin/ochttp"
)

var log = logging.Logger("mock main")

func MockMain(ctx context.Context, validateMiner []address.Address, cfg *types.Config) (string, []byte, error) {
	requestCfg := &types.RequestConfig{
		RequestQueueSize: 30,
		RequestTimeout:   time.Minute * 5,
		ClearInterval:    time.Minute * 5,
	}

	log.Infof("venus-gateway current version %s, listen %s", version.UserVersion, cfg.Listen)

	cli, _ := jwtclient.NewAuthClient(cfg.AuthUrl)

	minerValidator := validator.MockAuthMinerValidator{ValidatedAddr: validateMiner}

	walletStream := walletevent.NewWalletEventStream(ctx, cli, requestCfg)

	proofStream := proofevent.NewProofEventStream(ctx, minerValidator, requestCfg)
	marketStream := marketevent.NewMarketEventStream(ctx, minerValidator, &types.RequestConfig{
		RequestQueueSize: 30,
		RequestTimeout:   time.Hour * 12,
		ClearInterval:    time.Hour,
	})

	gatewayAPIImpl := api.NewGatewayAPIImpl(proofStream, walletStream, marketStream)

	log.Info("Setting up control endpoint at " + cfg.Listen)

	var fullNode sharedGatewayApiV1.IGatewayStruct
	permission.PermissionProxy(gatewayAPIImpl, &fullNode)
	gatewayAPI := (sharedGatewayApiV1.IGateway)(&fullNode)

	if len(cfg.RateLimitRedis) > 0 {
		limiter, err := ratelimit.NewRateLimitHandler(cfg.RateLimitRedis, nil,
			&jwtclient.ValueFromCtx{},
			jwtclient.WarpLimitFinder(cli),
			logging.Logger("rate-limit"))
		_ = logging.SetLogLevel("rate-limit", "info")
		if err != nil {
			return "", nil, err
		}
		var rateLimitAPI sharedGatewayApiV1.IGatewayStruct
		limiter.ProxyLimitFullAPI(gatewayAPI, &rateLimitAPI)
		gatewayAPI = &rateLimitAPI
	}

	mux := mux.NewRouter()
	//v1api
	rpcServerv1 := jsonrpc.NewServer()
	rpcServerv1.Register("Gateway", gatewayAPI)
	mux.Handle("/rpc/v1", rpcServerv1)

	//v0api
	v0FullNode := api.WrapperV1Full{IGateway: gatewayAPI}
	rpcServerv0 := jsonrpc.NewServer()
	rpcServerv0.Register("Gateway", v0FullNode)
	mux.Handle("/rpc/v0", rpcServerv0)

	mux.PathPrefix("/").Handler(http.DefaultServeMux)

	localJwt, err := utils.NewLocalJwtClient(".")
	if err != nil {
		return "", nil, err
	}
	handler := (http.Handler)(jwtclient.NewAuthMux(
		localJwt, jwtclient.WarpIJwtAuthClient(cli),
		mux, logging.Logger("Auth")))

	var tCnf = &metrics.TraceConfig{}
	var proxy, sampler, serverName = strings.TrimSpace(cfg.JaegerProxy),
		cfg.TraceSampler,
		strings.TrimSpace(cfg.TraceNodeName)

	if tCnf.JaegerTracingEnabled = len(proxy) != 0; tCnf.JaegerTracingEnabled {
		tCnf.ProbabilitySampler, tCnf.JaegerEndpoint, tCnf.ServerName =
			sampler, proxy, serverName
	}
	if repoter, err := metrics.RegisterJaeger(tCnf.ServerName, tCnf); err != nil {
		log.Fatalf("register %s JaegerRepoter to %s failed:%s", tCnf.ServerName, tCnf.JaegerEndpoint)
	} else if repoter != nil {
		log.Infof("register jaeger-tracing exporter to %s, with node-name:%s", tCnf.JaegerEndpoint, tCnf.ServerName)
		defer metrics.UnregisterJaeger(repoter)
		handler = &ochttp.Handler{Handler: handler}
	}

	srv := httptest.NewServer(handler)
	return srv.URL, localJwt.Token, nil
}
