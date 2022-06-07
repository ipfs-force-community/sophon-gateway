package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ipfs-force-community/venus-gateway/utils"

	"github.com/ipfs-force-community/venus-gateway/validator"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/gorilla/mux"
	logging "github.com/ipfs/go-log/v2"
	multiaddr "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/plugin/ochttp"

	"github.com/ipfs-force-community/metrics/ratelimit"

	"github.com/filecoin-project/venus-auth/cmd/jwtclient"

	"github.com/filecoin-project/venus/venus-shared/api/permission"
	"github.com/ipfs-force-community/metrics"
	"github.com/ipfs-force-community/venus-gateway/api"
	v0api "github.com/ipfs-force-community/venus-gateway/api/v0"
	"github.com/ipfs-force-community/venus-gateway/cmds"
	"github.com/ipfs-force-community/venus-gateway/marketevent"
	"github.com/ipfs-force-community/venus-gateway/proofevent"
	"github.com/ipfs-force-community/venus-gateway/types"
	"github.com/ipfs-force-community/venus-gateway/version"
	"github.com/ipfs-force-community/venus-gateway/walletevent"

	_ "github.com/filecoin-project/venus/pkg/crypto/bls"
	_ "github.com/filecoin-project/venus/pkg/crypto/secp"
)

var log = logging.Logger("main")

func main() {
	_ = logging.SetLogLevel("*", "INFO")

	app := &cli.App{
		Name:  "venus-gateway",
		Usage: "venus-gateway for proxy incoming wallet and proof",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "listen",
				Usage: "host address and port the worker api will listen on",
				Value: "/ip4/127.0.0.1/tcp/45132",
			},
		},
		Commands: []*cli.Command{
			runCmd, cmds.MinerCmds, cmds.WalletCmds, cmds.MarketCmds,
		},
	}
	app.Version = version.UserVersion
	if err := app.Run(os.Args); err != nil {
		log.Warn(err)
		os.Exit(1)
	}
}

var runCmd = &cli.Command{
	Name:  "run",
	Usage: "start venus-gateway daemon",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "auth-url", Usage: "venus auth url", Required: true},
		&cli.StringFlag{Name: "jaeger-proxy", EnvVars: []string{"VENUS_GATEWAY_JAEGER_PROXY"}},
		&cli.Float64Flag{Name: "trace-sampler", EnvVars: []string{"VENUS_GATEWAY_TRACE_SAMPLER"}, Value: 1.0},
		&cli.StringFlag{Name: "trace-node-name", Value: "venus-gateway"},
		&cli.StringFlag{Name: "rate-limit-redis", Hidden: true},
	},
	Action: func(cctx *cli.Context) error {
		cfg2 := &types.Config{
			Listen:         cctx.String("listen"),
			AuthUrl:        cctx.String("auth-url"),
			JaegerProxy:    cctx.String("jaeger-proxy"),
			TraceSampler:   cctx.Float64("trace-sampler"),
			TraceNodeName:  cctx.String("trace-node-name"),
			RateLimitRedis: cctx.String("rate-limit-redis"),
		}
		return RunMain(cctx.Context, cfg2)
	},
}

func RunMain(ctx context.Context, cfg *types.Config) error {
	requestCfg := types.DefaultConfig()

	log.Infof("venus-gateway current version %s, listen %s", version.UserVersion, cfg.Listen)

	cli, _ := jwtclient.NewAuthClient(cfg.AuthUrl)

	minerValidator := validator.NewMinerValidator(cli)

	walletStream := walletevent.NewWalletEventStream(ctx, cli, requestCfg)

	proofStream := proofevent.NewProofEventStream(ctx, minerValidator, requestCfg)
	marketStream := marketevent.NewMarketEventStream(ctx, minerValidator, &types.RequestConfig{
		RequestQueueSize: 30,
		RequestTimeout:   time.Hour * 7, //wait seven hour to do unseal
		ClearInterval:    time.Minute * 5,
	})

	gatewayAPIImpl := api.NewGatewayAPIImpl(proofStream, walletStream, marketStream)

	log.Info("Setting up control endpoint at " + cfg.Listen)

	var fullNode api.GatewayFullNodeStruct
	permission.PermissionProxy(gatewayAPIImpl, &fullNode)
	gatewayAPI := (api.GatewayFullNode)(&fullNode)

	if len(cfg.RateLimitRedis) > 0 {
		limiter, err := ratelimit.NewRateLimitHandler(cfg.RateLimitRedis, nil,
			&jwtclient.ValueFromCtx{},
			jwtclient.WarpLimitFinder(cli),
			logging.Logger("rate-limit"))
		_ = logging.SetLogLevel("rate-limit", "info")
		if err != nil {
			return err
		}
		var rateLimitAPI api.GatewayFullNodeStruct
		limiter.ProxyLimitFullAPI(gatewayAPI, &rateLimitAPI)
		gatewayAPI = &rateLimitAPI
	}

	mux := mux.NewRouter()
	//v1api
	rpcServerv1 := jsonrpc.NewServer()
	rpcServerv1.Register("Gateway", gatewayAPI)
	mux.Handle("/rpc/v1", rpcServerv1)

	//v0api
	v0FullNode := v0api.WrapperV1Full{GatewayFullNode: gatewayAPI}
	rpcServerv0 := jsonrpc.NewServer()
	rpcServerv0.Register("Gateway", v0FullNode)
	mux.Handle("/rpc/v0", rpcServerv0)

	mux.PathPrefix("/").Handler(http.DefaultServeMux)

	localJwt, err := utils.NewLocalJwtClient(".")
	if err != nil {
		return fmt.Errorf("make token failed:%s", err.Error())
	}
	err = localJwt.SaveToken()
	if err != nil {
		return err
	}

	handler := (http.Handler)(jwtclient.NewAuthMux(localJwt, jwtclient.WarpIJwtAuthClient(cli), mux))

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
	httptest.NewServer(handler)
	srv := &http.Server{Handler: handler}

	sigCh := make(chan os.Signal, 2)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		select {
		case sig := <-sigCh:
			log.Warnw("received shutdown", "signal", sig)
		case <-ctx.Done():
			log.Warn("received shutdown")
		}

		log.Info("Shutting down...")
		if err := srv.Shutdown(context.TODO()); err != nil {
			log.Errorf("shutting down RPC server failed: %s", err)
		}
	}()
	addr, err := multiaddr.NewMultiaddr(cfg.Listen)
	if err != nil {
		return err
	}

	nl, err := manet.Listen(addr)
	if err != nil {
		return err
	}

	log.Infof("start to rpc listen %s", nl.Addr())
	if err = srv.Serve(manet.NetListener(nl)); err != nil && err != http.ErrServerClosed {
		return err
	}

	log.Info("Graceful shutdown successful")
	return nil
}
