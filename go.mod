module github.com/ipfs-force-community/venus-gateway

go 1.16

require (
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-state-types v0.1.0
	github.com/filecoin-project/specs-actors v0.9.13
	github.com/filecoin-project/specs-actors/v3 v3.1.1 // indirect
	github.com/filecoin-project/venus-auth v1.1.1-0.20210601062027-260b83ff0191 // indirect
	github.com/filecoin-project/venus-wallet v1.1.0
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.7.4
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-log/v2 v2.1.3
	github.com/multiformats/go-multiaddr v0.3.1 // indirect
	github.com/urfave/cli/v2 v2.3.0
	golang.org/x/exp v0.0.0-20200513190911-00229845015e
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gopkg.in/resty.v1 v1.12.0
)

replace github.com/filecoin-project/go-jsonrpc => github.com/ipfs-force-community/go-jsonrpc v0.1.4-0.20210525064210-3d0a180a90b4
