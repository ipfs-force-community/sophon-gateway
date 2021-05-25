package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/filecoin-project/go-address"
	"github.com/urfave/cli/v2"
)

var MinerCmds = &cli.Command{
	Name:        "miner",
	Usage:       "miner cmds",
	Subcommands: []*cli.Command{listMinerCmds, getMinerStateCmds},
}

var listMinerCmds = &cli.Command{
	Name:  "list",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		api, closer, err := NewGatewayClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		miners, err := api.ListConnectedMiners(cctx.Context)
		if err != nil {
			return err
		}
		for _, minerAddr := range miners {
			fmt.Println(minerAddr)
		}
		return nil
	},
}

var getMinerStateCmds = &cli.Command{
	Name:      "state",
	Flags:     []cli.Flag{},
	ArgsUsage: "miner-addr",
	Action: func(cctx *cli.Context) error {
		api, closer, err := NewGatewayClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		mAddr, err := address.NewFromString(cctx.Args().Get(0))
		if err != nil {
			return err
		}
		minerState, err := api.ListMinerConnection(cctx.Context, mAddr)
		if err != nil {
			return err
		}
		minersBytes, err := json.MarshalIndent(minerState, " ", "\t")
		if err != nil {
			return err
		}
		fmt.Println(string(minersBytes))
		return nil
	},
}
