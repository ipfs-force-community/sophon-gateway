package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/venus-auth/cmd/jwtclient"
	v1API "github.com/filecoin-project/venus/venus-shared/api/gateway/v1"
	"github.com/filecoin-project/venus/venus-shared/api/permission"
	"github.com/gorilla/mux"
	"github.com/ipfs-force-community/metrics"
	"github.com/ipfs-force-community/metrics/ratelimit"
	logging "github.com/ipfs/go-log/v2"
	"github.com/mitchellh/go-homedir"
	multiaddr "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/plugin/ochttp"

	_ "github.com/filecoin-project/venus/pkg/crypto/bls"
	_ "github.com/filecoin-project/venus/pkg/crypto/secp"

	"github.com/ipfs-force-community/venus-gateway/api"
	"github.com/ipfs-force-community/venus-gateway/cmds"
	"github.com/ipfs-force-community/venus-gateway/config"
	"github.com/ipfs-force-community/venus-gateway/marketevent"
	"github.com/ipfs-force-community/venus-gateway/proofevent"
	"github.com/ipfs-force-community/venus-gateway/types"
	"github.com/ipfs-force-community/venus-gateway/utils"
	"github.com/ipfs-force-community/venus-gateway/validator"
	"github.com/ipfs-force-community/venus-gateway/version"
	"github.com/ipfs-force-community/venus-gateway/walletevent"
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
			&cli.StringFlag{
				Name:    "repo",
				Value:   "~/.venusgateway",
				EnvVars: []string{"VENUS_GATEWAY"},
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
		&cli.StringFlag{Name: "auth-url", Usage: "venus auth url"},
		&cli.StringFlag{Name: "jaeger-proxy", EnvVars: []string{"VENUS_GATEWAY_JAEGER_PROXY"}},
		&cli.Float64Flag{Name: "trace-sampler", EnvVars: []string{"VENUS_GATEWAY_TRACE_SAMPLER"}, Value: 1.0},
		&cli.StringFlag{Name: "trace-node-name", Value: "venus-gateway"},
		&cli.StringFlag{Name: "rate-limit-redis", Hidden: true},
	},
	Action: func(cctx *cli.Context) error {
		cfg := config.DefaultConfig()

		repoPath, err := homedir.Expand(cctx.String("repo"))
		if err != nil {
			return err
		}
		hasRepo, err := hasRepo(repoPath)
		if err != nil {
			return err
		}
		if hasRepo {
			cfg, err = config.ReadConfig(filepath.Join(repoPath, config.ConfigFile))
			if err != nil {
				return err
			}
		}

		parseFlag(cctx, cfg)

		if !hasRepo {
			if err := os.MkdirAll(repoPath, 0755); err != nil {
				return err
			}
			if err := config.WriteConfig(filepath.Join(repoPath, config.ConfigFile), cfg); err != nil {
				return err
			}
		}

		return RunMain(cctx.Context, repoPath, cfg)
	},
}

func hasRepo(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !fi.IsDir() {
		return false, fmt.Errorf("%s is not a directory", path)
	}

	return true, nil
}

func parseFlag(cctx *cli.Context, cfg *config.Config) {
	if cctx.IsSet("listen") {
		cfg.API.ListenAddress = cctx.String("listen")
	}
	if cctx.IsSet("auth-url") {
		cfg.Auth.URL = cctx.String("auth-url")
	}
	if cctx.IsSet("jaeger-proxy") {
		cfg.Trace.JaegerEndpoint = strings.TrimSpace(cctx.String("jaeger-proxy"))
		cfg.Trace.JaegerTracingEnabled = true
	}
	if cctx.IsSet("trace-sampler") {
		cfg.Trace.ProbabilitySampler = cctx.Float64("trace-sampler")
	}
	if cctx.IsSet("trace-node-name") {
		cfg.Trace.ServerName = strings.TrimSpace(cctx.String("trace-node-name"))
	}
	if cctx.IsSet("rate-limit-redis") {
		cfg.RateLimit.Redis = cctx.String("rate-limit-redis")
	}
}

func RunMain(ctx context.Context, repoPath string, cfg *config.Config) error {
	requestCfg := types.DefaultConfig()

	cli, _ := jwtclient.NewAuthClient(cfg.Auth.URL)

	minerValidator := validator.NewMinerValidator(cli)

	walletStream := walletevent.NewWalletEventStream(ctx, cli, requestCfg)

	proofStream := proofevent.NewProofEventStream(ctx, minerValidator, requestCfg)
	marketStream := marketevent.NewMarketEventStream(ctx, minerValidator, &types.RequestConfig{
		RequestQueueSize: 30,
		RequestTimeout:   time.Hour * 7, //wait seven hour to do unseal
		ClearInterval:    time.Minute * 5,
	})

	gatewayAPIImpl := api.NewGatewayAPIImpl(proofStream, walletStream, marketStream)

	log.Infof("venus-gateway current version %s", version.UserVersion)
	log.Infof("Setting up control endpoint at %v", cfg.API.ListenAddress)

	var fullNode v1API.IGatewayStruct
	permission.PermissionProxy(gatewayAPIImpl, &fullNode)
	gatewayAPI := (v1API.IGateway)(&fullNode)

	if len(cfg.RateLimit.Redis) > 0 {
		limiter, err := ratelimit.NewRateLimitHandler(cfg.RateLimit.Redis, nil,
			&jwtclient.ValueFromCtx{},
			jwtclient.WarpLimitFinder(cli),
			logging.Logger("rate-limit"))
		_ = logging.SetLogLevel("rate-limit", "info")
		if err != nil {
			return err
		}
		var rateLimitAPI v1API.IGatewayStruct
		limiter.ProxyLimitFullAPI(gatewayAPI, &rateLimitAPI)
		gatewayAPI = &rateLimitAPI
	}

	mux := mux.NewRouter()
	//v1api
	rpcServerv1 := jsonrpc.NewServer()
	rpcServerv1.Register("Gateway", gatewayAPI)
	mux.Handle("/rpc/v1", rpcServerv1)

	v0FullNode := api.WrapperV1Full{IGateway: gatewayAPI}
	rpcServerv0 := jsonrpc.NewServer()
	rpcServerv0.Register("Gateway", v0FullNode)
	mux.Handle("/rpc/v0", rpcServerv0)

	mux.PathPrefix("/").Handler(http.DefaultServeMux)

	localJwt, err := utils.NewLocalJwtClient(repoPath)
	if err != nil {
		return fmt.Errorf("make token failed:%s", err.Error())
	}
	err = localJwt.SaveToken()
	if err != nil {
		return err
	}

	handler := (http.Handler)(jwtclient.NewAuthMux(localJwt, jwtclient.WarpIJwtAuthClient(cli), mux))

	log.Infof("trace config %+v", cfg.Trace)
	repoter, err := metrics.RegisterJaeger(cfg.Trace.ServerName, cfg.Trace)
	if err != nil {
		return fmt.Errorf("register jaeger exporter failed %v", cfg.Trace)
	}
	if repoter != nil {
		log.Info("register jaeger exporter success!")

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
	addr, err := multiaddr.NewMultiaddr(cfg.API.ListenAddress)
	if err != nil {
		return err
	}

	nl, err := manet.Listen(addr)
	if err != nil {
		return err
	}

	if err = srv.Serve(manet.NetListener(nl)); err != nil && err != http.ErrServerClosed {
		return err
	}

	log.Info("Graceful shutdown successful")
	return nil
}
