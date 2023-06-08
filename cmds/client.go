package cmds

import (
	"io/ioutil"
	"path/filepath"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"

	_ "github.com/filecoin-project/venus/venus-shared/api"
	v2API "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"

	"github.com/ipfs-force-community/sophon-gateway/config"
)

func NewGatewayClient(ctx *cli.Context) (v2API.IGateway, jsonrpc.ClientCloser, error) {
	repoPath, err := homedir.Expand(ctx.String("repo"))
	if err != nil {
		return nil, nil, err
	}

	cfg, err := config.ReadConfig(filepath.Join(repoPath, config.ConfigFile))
	if err != nil {
		return nil, nil, err
	}

	listen := ctx.String("listen")
	if !ctx.IsSet("listen") {
		listen = cfg.API.ListenAddress
	}

	token, err := ioutil.ReadFile(filepath.Join(repoPath, "token"))
	if err != nil {
		return nil, nil, err
	}

	return v2API.DialIGatewayRPC(ctx.Context, listen, string(token), nil)
}
