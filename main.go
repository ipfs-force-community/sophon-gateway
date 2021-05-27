package main

import (
	"context"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/gorilla/mux"
	"github.com/ipfs-force-community/venus-gateway/cmds"
	"github.com/ipfs-force-community/venus-gateway/proofevent"
	"github.com/ipfs-force-community/venus-gateway/types"
	"github.com/ipfs-force-community/venus-gateway/version"
	"github.com/ipfs-force-community/venus-gateway/walletevent"
	logging "github.com/ipfs/go-log/v2"
	multiaddr "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr/net"
	"github.com/urfave/cli/v2"
	"net/http"
	"os"
	"time"
)

var log = logging.Logger("main")

func main() {
	logging.SetLogLevel("*", "INFO")

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
		&cli.StringFlag{
			Name:  "auth-url",
			Usage: "venus auth url",
		},
	},
	Action: func(cctx *cli.Context) error {
		ctx := cctx.Context
		mux := mux.NewRouter()
		address := cctx.String("listen")
		cfg := &types.Config{
			RequestQueueSize: 30,
			RequestTimeout:   time.Minute * 5,
		}
		proofStream := proofevent.NewProofEventStream(ctx, cfg)
		walletStream := walletevent.NewWalletEventStream(ctx, cfg)
		gatewayAPI := NewGatewayAPI(proofStream, walletStream)
		log.Info("Setting up control endpoint at " + address)
		rpcServer := jsonrpc.NewServer(func(c *jsonrpc.ServerConfig) {
		})
		rpcServer.Register("Filecoin", gatewayAPI)
		mux.Handle("/rpc/v0", rpcServer)
		mux.PathPrefix("/").Handler(http.DefaultServeMux) // pprof
		cli := NewJWTClient(cctx.String("auth-url"))
		srv := &http.Server{
			Handler: &VenusAuthHandler{
				Verify: cli.Verify,
				Next:   mux.ServeHTTP,
			},
		}
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
