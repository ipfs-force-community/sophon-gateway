package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/etherlabsio/healthcheck/v2"

	"github.com/gorilla/mux"
	logging "github.com/ipfs/go-log/v2"
	"github.com/mitchellh/go-homedir"
	multiaddr "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/plugin/ochttp"

	"github.com/filecoin-project/go-jsonrpc"

	"github.com/ipfs-force-community/sophon-auth/core"
	"github.com/ipfs-force-community/sophon-auth/jwtclient"

	_ "github.com/filecoin-project/venus/pkg/crypto/bls"
	_ "github.com/filecoin-project/venus/pkg/crypto/delegated"
	_ "github.com/filecoin-project/venus/pkg/crypto/secp"
	v2API "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"
	"github.com/filecoin-project/venus/venus-shared/api/permission"

	"github.com/ipfs-force-community/metrics"
	"github.com/ipfs-force-community/metrics/ratelimit"

	"github.com/ipfs-force-community/sophon-gateway/api"
	"github.com/ipfs-force-community/sophon-gateway/api/v1api"
	"github.com/ipfs-force-community/sophon-gateway/cluster"
	"github.com/ipfs-force-community/sophon-gateway/cmds"
	"github.com/ipfs-force-community/sophon-gateway/config"
	"github.com/ipfs-force-community/sophon-gateway/marketevent"
	metrics2 "github.com/ipfs-force-community/sophon-gateway/metrics"
	"github.com/ipfs-force-community/sophon-gateway/proofevent"
	"github.com/ipfs-force-community/sophon-gateway/proxy"
	"github.com/ipfs-force-community/sophon-gateway/types"
	"github.com/ipfs-force-community/sophon-gateway/validator"
	"github.com/ipfs-force-community/sophon-gateway/version"
	"github.com/ipfs-force-community/sophon-gateway/walletevent"
)

const (
	oldRepoPath = "~/.venusgateway"
	defRepoPath = "~/.sophon-gateway"
)

var log = logging.Logger("main")

func main() {
	_ = logging.SetLogLevel("*", "INFO")

	app := &cli.App{
		Name:  "sophon-gateway",
		Usage: "sophon-gateway for proxy incoming wallet and proof",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "listen",
				Usage: "host address and port the worker api will listen on",
				Value: "/ip4/127.0.0.1/tcp/45132",
			},
			&cli.StringFlag{
				Name:    "repo",
				Value:   defRepoPath,
				EnvVars: []string{"SOPHON_GATEWAY"},
			},
		},
		Commands: []*cli.Command{
			runCmd, cmds.MinerCmds, cmds.WalletCmds, cmds.MarketCmds, cmds.ProxyCmds,
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
	Usage: "start sophon-gateway daemon",
	Flags: []cli.Flag{
		&cli.StringFlag{Name: "auth-url", Usage: "sophon auth url"},
		&cli.StringFlag{Name: "auth-token", Usage: "sophon auth token"},
		&cli.StringFlag{Name: "jaeger-proxy", EnvVars: []string{"SOPHON_GATEWAY_JAEGER_PROXY"}},
		&cli.Float64Flag{Name: "trace-sampler", EnvVars: []string{"SOPHON_GATEWAY_TRACE_SAMPLER"}, Value: 1.0},
		&cli.StringFlag{Name: "trace-node-name", Value: "sophon-gateway"},
		&cli.StringFlag{Name: "rate-limit-redis", Hidden: true},
	},
	Action: func(cctx *cli.Context) error {
		cfg := config.DefaultConfig()

		repoPath, err := homedir.Expand(cctx.String("repo"))
		if err != nil {
			return err
		}
		// todo: remove compatibility code
		repoPath, err = cmds.GetRepoPath(repoPath)
		if err != nil {
			return err
		}

		hasRepo, err := cmds.HasRepo(repoPath)
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
			if err := os.MkdirAll(repoPath, 0o755); err != nil {
				return err
			}
			if err := config.WriteConfig(filepath.Join(repoPath, config.ConfigFile), cfg); err != nil {
				return err
			}
		}

		return RunMain(cctx.Context, repoPath, cfg)
	},
}

func parseFlag(cctx *cli.Context, cfg *config.Config) {
	if cctx.IsSet("listen") {
		cfg.API.ListenAddress = cctx.String("listen")
	}
	if cctx.IsSet("auth-url") {
		cfg.Auth.URL = cctx.String("auth-url")
	}
	if cctx.IsSet("auth-token") {
		cfg.Auth.Token = cctx.String("auth-token")
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

	remoteJwtCli, err := jwtclient.NewAuthClient(cfg.Auth.URL, cfg.Auth.Token)
	if err != nil {
		return err
	}

	minerValidator := validator.NewMinerValidator(remoteJwtCli)

	walletStream := walletevent.NewWalletEventStream(ctx, remoteJwtCli, requestCfg)

	proofStream := proofevent.NewProofEventStream(ctx, minerValidator, requestCfg)
	marketStream := marketevent.NewMarketEventStream(ctx, minerValidator, &types.RequestConfig{
		RequestQueueSize: 30,
		RequestTimeout:   time.Hour * 7, // wait seven hour to do unseal
		ClearInterval:    time.Minute * 5,
	})

	listenAddressForCluster := ""
	if cfg.Cluster != nil {
		listenAddressForCluster = cfg.Cluster.ListenAddress
	}
	cluster, err := cluster.NewCluster(ctx, cfg.API.ListenAddress, listenAddressForCluster, cfg.Auth.Token)
	if err != nil {
		return err
	}

	chainServiceProxy := proxy.NewProxy()

	gatewayAPIImpl := api.NewGatewayAPIImpl(proofStream, walletStream, marketStream, chainServiceProxy, cluster)

	log.Infof("sophon-gateway current version %s", version.UserVersion)
	log.Infof("Setting up control endpoint at %v", cfg.API.ListenAddress)

	var fullNode v2API.IGatewayStruct
	permission.PermissionProxy(gatewayAPIImpl, &fullNode)
	gatewayAPI := (v2API.IGateway)(&fullNode)

	if len(cfg.RateLimit.Redis) > 0 {
		limiter, err := ratelimit.NewRateLimitHandler(cfg.RateLimit.Redis, nil,
			&core.ValueFromCtx{},
			jwtclient.WarpLimitFinder(remoteJwtCli),
			logging.Logger("rate-limit"))
		_ = logging.SetLogLevel("rate-limit", "info")
		if err != nil {
			return err
		}
		var rateLimitAPI v2API.IGatewayStruct
		limiter.ProxyLimitFullAPI(gatewayAPI, &rateLimitAPI)
		gatewayAPI = &rateLimitAPI
	}

	mux := mux.NewRouter()

	// v2api(newest api)
	rpcServerV2 := jsonrpc.NewServer()
	rpcServerV2.Register("Gateway", gatewayAPI)
	mux.Handle("/rpc/v2", rpcServerV2)

	lowerFullNode := v1api.WrapperV2Full{IGateway: gatewayAPI}
	rpcServerV1 := jsonrpc.NewServer()
	rpcServerV1.Register("Gateway", lowerFullNode)
	mux.Handle("/rpc/v1", rpcServerV1)

	mux.PathPrefix("/").Handler(http.DefaultServeMux)

	localJwtCli, localToken, err := jwtclient.NewLocalAuthClient()
	if err != nil {
		return fmt.Errorf("failed to generate local jwt client: %v", err)
	}
	// save local token to token file
	err = ioutil.WriteFile(path.Join(repoPath, "token"), localToken, 0o644)
	if err != nil {
		return fmt.Errorf("failed to save local token to token file: %w", err)
	}

	authMux := jwtclient.NewAuthMux(localJwtCli, jwtclient.WarpIJwtAuthClient(remoteJwtCli), mux)
	authMux.TrustHandle("/debug/pprof/", http.DefaultServeMux)
	authMux.TrustHandle("/healthcheck", healthcheck.Handler())

	if err := metrics2.SetupMetrics(ctx, cfg.Metrics, gatewayAPIImpl); err != nil {
		return err
	}
	handler := (http.Handler)(authMux)
	if cfg.Trace.JaegerTracingEnabled {
		log.Infof("trace config %+v", cfg.Trace)
		reporter, err := metrics.SetupJaegerTracing(cfg.Trace.ServerName, cfg.Trace)
		if err != nil {
			return fmt.Errorf("register jaeger exporter failed %v", cfg.Trace)
		}

		if reporter != nil {
			log.Info("register jaeger exporter success!")

			defer func() {
				err := metrics.ShutdownJaeger(ctx, reporter)
				if err != nil {
					log.Errorf("shutdown jaeger failed: %s", err)
				}
			}()
			handler = &ochttp.Handler{Handler: handler}
		}
	}

	err = chainServiceProxy.RegisterReverseByAddr(proxy.HostAuth, cfg.Auth.URL)
	if err != nil {
		return err
	}
	if cfg.Node != nil {
		err := chainServiceProxy.RegisterReverseByAddr(proxy.HostNode, *cfg.Node)
		if err != nil {
			return err
		}
	}
	if cfg.Messager != nil {
		err := chainServiceProxy.RegisterReverseByAddr(proxy.HostMessager, *cfg.Messager)
		if err != nil {
			return err
		}
	}
	if cfg.Miner != nil {
		err := chainServiceProxy.RegisterReverseByAddr(proxy.HostMiner, *cfg.Miner)
		if err != nil {
			return err
		}
	}
	if cfg.Droplet != nil {
		err := chainServiceProxy.RegisterReverseByAddr(proxy.HostDroplet, *cfg.Droplet)
		if err != nil {
			return err
		}
	}
	chainServiceProxy.RegisterReverseHandler(proxy.HostGateway, handler)

	handler = chainServiceProxy.ProxyMiddleware(handler)

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
