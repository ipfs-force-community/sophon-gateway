package cmds

import (
	"io/ioutil"
	"path/filepath"

	"github.com/filecoin-project/go-jsonrpc"
	_ "github.com/filecoin-project/venus/venus-shared/api"
	v1API "github.com/filecoin-project/venus/venus-shared/api/gateway/v1"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"

	"github.com/ipfs-force-community/venus-gateway/config"
	"github.com/ipfs-force-community/venus-gateway/utils"
)

func NewGatewayClient(ctx *cli.Context) (v1API.IGateway, jsonrpc.ClientCloser, error) {
	repoPath, err := homedir.Expand(ctx.String("repo"))
	if err != nil {
		return nil, nil, err
	}

	listen := ctx.String("listen")
	if !ctx.IsSet("listen") {
		cfg, err := config.ReadConfig(filepath.Join(repoPath, config.ConfigFile))
		if err != nil {
			return nil, nil, err
		}
		listen = cfg.API.ListenAddress
	}

	token, err := ioutil.ReadFile(filepath.Join(repoPath, utils.TokenFile))
	if err != nil {
		return nil, nil, err
	}

	return v1API.DialIGatewayRPC(ctx.Context, listen, string(token), nil)
}
