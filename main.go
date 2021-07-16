package main

import (
	"context"
	"github.com/ipfs-force-community/metrics/ratelimit"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/venus-auth/cmd/jwtclient"
	"github.com/gorilla/mux"
	"github.com/ipfs-force-community/metrics"
	"github.com/ipfs-force-community/venus-gateway/api"
	"github.com/ipfs-force-community/venus-gateway/cmds"
	"github.com/ipfs-force-community/venus-gateway/proofevent"
	"github.com/ipfs-force-community/venus-gateway/types"
	"github.com/ipfs-force-community/venus-gateway/version"
	"github.com/ipfs-force-community/venus-gateway/walletevent"
	logging "github.com/ipfs/go-log/v2"
	multiaddr "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"go.opencensus.io/plugin/ochttp"
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
			runCmd, cmds.MinerCmds, cmds.WalletCmds,
		},
	}
	app.Version = version.UserVersion + "--" + version.CurrentCommit
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
		&cli.StringFlag{Name: "jaeger-proxy", EnvVars: []string{"VENUS_GATEWAY_JAEGER_PROXY"}, Hidden: true},
		&cli.Float64Flag{Name: "trace-sampler", EnvVars: []string{"VENUS_GATEWAY_TRACE_SAMPLER"}, Value: 1.0, Hidden: true},
		&cli.StringFlag{Name: "trace-node-name", Value: "venus-gateway", Hidden: true},
		&cli.StringFlag{Name: "rate-limit-redis", Hidden: true},
	},
	Before: func(c *cli.Context) error {
		var mCnf = &metrics.TraceConfig{}

		var proxy, sampler, serverName = strings.TrimSpace(c.String("jaeger-proxy")),
			c.Float64("trace-sampler"),
			strings.TrimSpace(c.String("trace-node-name"))

		if mCnf.JaegerTracingEnabled = len(proxy) != 0; mCnf.JaegerTracingEnabled {
			mCnf.ProbabilitySampler, mCnf.JaegerEndpoint, mCnf.ServerName =
				sampler, proxy, serverName
		}

		c.Context = context.WithValue(c.Context, "trace-config", mCnf)
		return nil
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context

		address := cctx.String("listen")
		cfg := &types.Config{
			RequestQueueSize: 30,
			RequestTimeout:   time.Minute * 5,
		}
		cli := jwtclient.NewJWTClient(cctx.String("auth-url"))

		proofStream := proofevent.NewProofEventStream(ctx, cli, cfg)
		walletStream := walletevent.NewWalletEventStream(ctx, cli, cfg)

		gatewayAPI := NewGatewayAPI(proofStream, walletStream)

		log.Info("Setting up control endpoint at " + address)

		fullAPI := &api.FullStruct{}
		api.PermissionProxy(gatewayAPI, fullAPI)

		rpcServer := jsonrpc.NewServer(jsonrpc.WithProxyBind(jsonrpc.PBField))
		if cctx.IsSet("rate-limit-redis") {
			limiter, err := ratelimit.NewRateLimitHandler(cctx.String("rate-limit-redis"), nil,
				&jwtclient.ValueFromCtx{},
				jwtclient.WarpLimitFinder(cli),
				logging.Logger("rate-limit"))
			_ = logging.SetLogLevel("rate-limit", "info")
			if err != nil {
				return err
			}
			var rateLimitAPI api.FullStruct
			limiter.WarperLimiter(*fullAPI, &rateLimitAPI)
			fullAPI = &rateLimitAPI
		}

		rpcServer.Register("Gateway", fullAPI)

		mux := mux.NewRouter()
		mux.Handle("/rpc/v0", rpcServer)
		mux.PathPrefix("/").Handler(http.DefaultServeMux)

		handler := (http.Handler)(jwtclient.NewAuthMux(
			&localJwtClient{}, jwtclient.WarpIJwtAuthClient(cli),
			mux, logging.Logger("Auth")))

		tCnf := cctx.Context.Value("trace-config").(*metrics.TraceConfig)
		if repoter, err := metrics.RegisterJaeger(tCnf.ServerName, tCnf); err != nil {
			log.Fatalf("register %s JaegerRepoter to %s failed:%s", tCnf.ServerName, tCnf.JaegerEndpoint)
		} else if repoter != nil {
			log.Infof("register jaeger-tracing exporter to %s, with node-name:%s", tCnf.JaegerEndpoint, tCnf.ServerName)
			defer metrics.UnregisterJaeger(repoter)
			handler = &ochttp.Handler{Handler: handler}
		}

		srv := &http.Server{Handler: handler}

		sigCh := make(chan os.Signal, 2)
		go func() {
			select {
			case sig := <-sigCh:
				log.Warnw("received shutdown", "signal", sig)
			case <-ctx.Done():
				log.Warn("received shutdown")
			}

			log.Warn("Shutting down...")
			if err := srv.Shutdown(context.TODO()); err != nil {
				log.Errorf("shutting down RPC server failed: %s", err)
			}
			log.Warn("Graceful shutdown successful")
		}()

		addr, err := multiaddr.NewMultiaddr(address)
		if err != nil {
			return err
		}

		nl, err := manet.Listen(addr)
		if err != nil {
			return err
		}
		return srv.Serve(manet.NetListener(nl))
	},
}
