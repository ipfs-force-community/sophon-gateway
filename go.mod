module github.com/ipfs-force-community/venus-gateway

go 1.16

require (
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-state-types v0.1.1-0.20210506134452-99b279731c48
	github.com/filecoin-project/specs-actors/v5 v5.0.0-20210602024058-0c296bb386bf
	github.com/filecoin-project/venus-auth v1.1.1-0.20210601064545-55f3162444fd
	github.com/filecoin-project/venus-wallet v1.1.1-0.20210608022957-9c0291b2f6c2
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.7.4
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-log/v2 v2.1.3
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/urfave/cli/v2 v2.3.0
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
)

replace github.com/filecoin-project/go-jsonrpc => github.com/ipfs-force-community/go-jsonrpc v0.1.4-0.20210525064210-3d0a180a90b4
