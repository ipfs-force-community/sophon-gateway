module github.com/ipfs-force-community/venus-gateway

go 1.16

require (
	github.com/filecoin-project/go-address v0.0.6
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-state-types v0.1.3
	github.com/filecoin-project/specs-actors/v5 v5.0.4
	github.com/filecoin-project/specs-storage v0.2.0
	github.com/filecoin-project/venus v1.2.0-rc5.0.20220217131630-a1df3e59a16c
	github.com/filecoin-project/venus-auth v1.3.1-0.20210809053831-012d55d5f578
	github.com/gbrlsnchs/jwt/v3 v3.0.1
	github.com/gorilla/mux v1.8.0
	github.com/ipfs-force-community/metrics v1.0.1-0.20211228055608-9462dc86e157
	github.com/ipfs/go-cid v0.1.0
	github.com/ipfs/go-log/v2 v2.4.0
	github.com/modern-go/reflect2 v1.0.2
	github.com/multiformats/go-multiaddr v0.4.1
	github.com/pkg/errors v0.9.1
	github.com/urfave/cli/v2 v2.3.0
	go.opencensus.io v0.23.0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
)

replace github.com/filecoin-project/filecoin-ffi => ./extern/filecoin-ffi

replace github.com/ipfs/go-ipfs-cmds => github.com/ipfs-force-community/go-ipfs-cmds v0.6.1-0.20210521090123-4587df7fa0ab

replace github.com/ipfs-force-community/venus-gateway => ./

replace github.com/filecoin-project/go-jsonrpc => github.com/ipfs-force-community/go-jsonrpc v0.1.4-0.20210731021807-68e5207079bc
