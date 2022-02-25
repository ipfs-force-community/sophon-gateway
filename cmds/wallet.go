package cmds

import (
	"encoding/json"
	"fmt"

	types "github.com/filecoin-project/venus/venus-shared/types/gateway"
	"github.com/urfave/cli/v2"
)

var WalletCmds = &cli.Command{
	Name:        "wallet",
	Usage:       "wallet cmds",
	Subcommands: []*cli.Command{listWalletCmds, getWalletStateCmds, getWalletByAccountCmds},
}

var listWalletCmds = &cli.Command{
	Name:  "list",
	Flags: []cli.Flag{},
	Action: func(cctx *cli.Context) error {
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

var getWalletByAccountCmds = &cli.Command{
	Name:      "list-support",
	Usage:     "query which wallet support the account",
	Flags:     []cli.Flag{},
	ArgsUsage: "account",
	Action: func(cctx *cli.Context) error {
		api, closer, err := NewGatewayClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		wallets, err := api.ListWalletInfo(cctx.Context)
		if err != nil {
			return err
		}

		account := cctx.Args().Get(0)
		var supportWallets []*types.WalletDetail
		for _, wallet := range wallets {
			for _, supportAccount := range wallet.SupportAccounts {
				if supportAccount == account {
					supportWallets = append(supportWallets, wallet)
				}
			}
		}
		minersBytes, err := json.MarshalIndent(supportWallets, " ", "\t")
		if err != nil {
			return err
		}
		fmt.Println(string(minersBytes))
		return nil
	},
}
