package proxy

import (
	"fmt"

	"github.com/filecoin-project/venus/venus-shared/api"
	chainV0 "github.com/filecoin-project/venus/venus-shared/api/chain/v0"
	chainV1 "github.com/filecoin-project/venus/venus-shared/api/chain/v1"
	gatewayV0 "github.com/filecoin-project/venus/venus-shared/api/gateway/v0"
	marketV0 "github.com/filecoin-project/venus/venus-shared/api/market/v0"
	marketV1 "github.com/filecoin-project/venus/venus-shared/api/market/v1"
	messager "github.com/filecoin-project/venus/venus-shared/api/messager"
	authCore "github.com/ipfs-force-community/sophon-auth/core"
	minerClient "github.com/ipfs-force-community/sophon-miner/api/client"
)

// map head -> addr
// key -> header ; key -> addr

type HostKey string

const (
	HostUnknown  HostKey = ""
	HostMessager HostKey = "MESSAGER"
	HostDroplet  HostKey = "DROPLET"
	HostNode     HostKey = "VENUS"
	HostAuth     HostKey = "AUTH"
	HostMiner    HostKey = "MINER"
	HostGateway  HostKey = "GATEWAY"
)

const (
	VenusAPINamespaceHeader = api.VenusAPINamespaceHeader
	emptyHeaderValue        = ""
)

var (
	Header2HostPreset map[string]HostKey = map[string]HostKey{
		chainV1.APINamespace:     HostNode,
		chainV0.APINamespace:     HostNode,
		marketV0.APINamespace:    HostDroplet,
		marketV1.APINamespace:    HostDroplet,
		messager.APINamespace:    HostMessager,
		authCore.APINamespace:    HostAuth,
		minerClient.APINamespace: HostMiner,
		gatewayV0.APINamespace:   HostGateway,
		// use gateway by default
		emptyHeaderValue: HostGateway,
	}
)

var (
	ErrorInvalidHeader            = fmt.Errorf("invalid venus proxy header for %s", api.VenusAPINamespaceHeader)
	ErrorNoReverseProxyRegistered = fmt.Errorf("no reverse proxy registered")
)
