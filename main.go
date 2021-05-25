package main

import (
	"context"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/gorilla/mux"
	"github.com/ipfs-force-community/venus-gateway/cmds"
	"github.com/ipfs-force-community/venus-gateway/proofevent"
	"github.com/ipfs-force-community/venus-gateway/types"
	"github.com/ipfs-force-community/venus-gateway/walletevent"
	logging "github.com/ipfs/go-log/v2"
	"github.com/prometheus/common/log"
	"github.com/urfave/cli/v2"
	"net"
	"net/http"
	"os"
	"time"
)

func main() {
	logging.SetLogLevel("*", "INFO")

	app := &cli.App{
		Name:  "venus-gateway",
		Usage: "venus-gateway for proxy incoming wallet and proof",
		Flags: []cli.Flag{},
		Commands: []*cli.Command{
			runCmd, cmds.MinerCmds, cmds.WalletCmds,
		},
	}

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
			Name:  "listen",
			Usage: "host address and port the worker api will listen on",
			Value: "127.0.0.1:45132",
		},
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
		proofStream := proofevent.NewProofEventStream(cfg)
		walletStream := walletevent.NewWalletEventStream(cfg)
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
		go func() {
			<-ctx.Done()
			log.Warn("Shutting down...")
			if err := srv.Shutdown(context.TODO()); err != nil {
				log.Errorf("shutting down RPC server failed: %s", err)
			}
			log.Warn("Graceful shutdown successful")
		}()

		nl, err := net.Listen("tcp", address)
		if err != nil {
			return err
		}
		return srv.Serve(nl)
	},
}
