package cmds

import (
	"fmt"

	"github.com/ipfs-force-community/sophon-gateway/proxy"
	"github.com/urfave/cli/v2"
)

var ProxyCmds = &cli.Command{
	Name:        "proxy",
	Usage:       "manipulate proxy registered in gateway",
	Subcommands: []*cli.Command{setProxyCmd},
}

var setProxyCmd = &cli.Command{
	Name:  "set",
	Usage: "set proxy (or unset proxy by setting a empty url)",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:     "type",
			Usage:    fmt.Sprintf("specify which type of venus component, to proxy, e.g. %s, %s, %s, %s, %s", proxy.HostAuth, proxy.HostNode, proxy.HostMessager, proxy.HostMiner, proxy.HostDroplet),
			Required: true,
		},
		&cli.StringFlag{
			Name:  "url",
			Usage: "the url will redirect to",
		},
	},
	Action: func(cctx *cli.Context) error {
		api, closer, err := NewGatewayClient(cctx)
		if err != nil {
			return err
		}
		defer closer()

		u := cctx.String("url")

		t := proxy.HostKey(cctx.String("type"))
		if t != proxy.HostAuth && t != proxy.HostNode && t != proxy.HostMessager && t != proxy.HostMiner && t != proxy.HostDroplet {
			return fmt.Errorf("invalid type %s", t)
		}

		err = api.RegisterReverse(cctx.Context, t, u)
		if err != nil {
			return err
		}

		if u == "" {
			fmt.Printf("unset %s success \n", t)
			return nil
		}

		fmt.Printf("set %s to %s success \n", t, u)
		return nil
	},
}
