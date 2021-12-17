package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/urfave/cli/v2"
)

var MarketCmds = &cli.Command{
	Name:        "market",
	Usage:       "market cmds",
	Subcommands: []*cli.Command{listMarketCmd},
}

var listMarketCmd = &cli.Command{
	Name:  "list",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		api, closer, err := NewGatewayClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		marketConnState, err := api.ListMarketConnectionsState(cctx.Context)
		if err != nil {
			return err
		}
		connBytes, err := json.MarshalIndent(marketConnState, " ", "\t")
		if err != nil {
			return err
		}
		fmt.Println(string(connBytes))
		return nil
	},
}
