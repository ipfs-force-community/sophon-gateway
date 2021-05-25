package cmds

import (
	"encoding/json"
	"fmt"
	"github.com/urfave/cli/v2"
)

var WalletCmds = &cli.Command{
	Name:        "wallet",
	Usage:       "wallet cmds",
	Subcommands: []*cli.Command{listWalletCmds, getWalletStateCmds},
}

var listWalletCmds = &cli.Command{
	Name:  "list",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
		fmt.Print("xxxxxx")
		api, closer, err := NewGatewayClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		wallets, err := api.ListWalletInfo(cctx.Context)
		if err != nil {
			return err
		}
		minersBytes, err := json.MarshalIndent(wallets, " ", "\t")
		if err != nil {
			return err
		}
		fmt.Println(string(minersBytes))
		return nil
	},
}

var getWalletStateCmds = &cli.Command{
	Name:      "state",
	Flags:     []cli.Flag{},
	ArgsUsage: "wallet-account",
	Action: func(cctx *cli.Context) error {
		api, closer, err := NewGatewayClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		walletAccount := cctx.Args().Get(0)
		walletState, err := api.ListWalletInfoByWallet(cctx.Context, walletAccount)
		if err != nil {
			return err
		}
		walletStateBytes, err := json.MarshalIndent(walletState, " ", "\t")
		if err != nil {
			return err
		}
		fmt.Println(string(walletStateBytes))
		return nil
	},
}
