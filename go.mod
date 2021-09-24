module github.com/ipfs-force-community/venus-gateway

go 1.16

require (
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-state-types v0.1.1-0.20210722133031-ad9bfe54c124
	github.com/filecoin-project/specs-actors/v5 v5.0.1
	github.com/filecoin-project/specs-storage v0.1.1-0.20201105051918-5188d9774506
	github.com/filecoin-project/venus v1.0.4
	github.com/filecoin-project/venus-auth v1.2.2-0.20210721103851-593a379c4916
	github.com/gbrlsnchs/jwt/v3 v3.0.0
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/ipfs-force-community/metrics v1.0.0
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-log/v2 v2.3.0
	github.com/modern-go/reflect2 v1.0.1
	github.com/multiformats/go-multiaddr v0.3.3
	github.com/pkg/errors v0.9.1
	github.com/urfave/cli/v2 v2.3.0
	go.opencensus.io v0.23.0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

replace github.com/ipfs/go-ipfs-cmds => github.com/ipfs-force-community/go-ipfs-cmds v0.6.1-0.20210521090123-4587df7fa0ab

replace github.com/ipfs-force-community/venus-gateway => ./

replace github.com/filecoin-project/go-jsonrpc => github.com/ipfs-force-community/go-jsonrpc v0.1.4-0.20210731021807-68e5207079bc
