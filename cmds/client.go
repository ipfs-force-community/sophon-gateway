package cmds

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/mitchellh/go-homedir"
	"github.com/urfave/cli/v2"

	_ "github.com/filecoin-project/venus/venus-shared/api"
	v2API "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"

	"github.com/ipfs-force-community/sophon-gateway/config"
)

const oldRepoPath = "~/.venusgateway"

func NewGatewayClient(ctx *cli.Context) (v2API.IGateway, jsonrpc.ClientCloser, error) {
	repoPath, err := homedir.Expand(ctx.String("repo"))
	if err != nil {
		return nil, nil, err
	}
	repoPath, err = GetRepoPath(repoPath)
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

func HasRepo(path string) (bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	if !fi.IsDir() {
		return false, fmt.Errorf("%s is not a directory", path)
	}

	return true, nil
}

func GetRepoPath(repoPath string) (string, error) {
	has, err := HasRepo(repoPath)
	if err != nil {
		return "", err
	}
	if !has {
		// check old repo path
		rPath, err := homedir.Expand(oldRepoPath)
		if err != nil {
			return "", err
		}
		has, err = HasRepo(rPath)
		if err != nil {
			return "", err
		}
		if has {
			return rPath, nil
		}
	}
	return repoPath, nil
}
