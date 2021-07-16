module github.com/ipfs-force-community/venus-gateway

go 1.16

require (
	github.com/cpuguy83/go-md2man/v2 v2.0.0 // indirect
	github.com/filecoin-project/go-address v0.0.5
	github.com/filecoin-project/go-jsonrpc v0.1.4-0.20210217175800-45ea43ac2bec
	github.com/filecoin-project/go-state-types v0.1.1-0.20210506134452-99b279731c48
	github.com/filecoin-project/specs-actors v0.9.14 // indirect
	github.com/filecoin-project/specs-actors/v5 v5.0.1
	github.com/filecoin-project/venus-auth v1.2.2-0.20210716071053-565a8999f9e5
	github.com/golang/snappy v0.0.2-0.20190904063534-ff6b7dc882cf // indirect
	github.com/google/flatbuffers v2.0.0+incompatible // indirect
	github.com/google/uuid v1.2.0
	github.com/gorilla/mux v1.8.0
	github.com/ipfs-force-community/metrics v0.0.0-20210716075100-f6c912bf4b47
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-ipfs-cmds v0.5.0 // indirect
	github.com/ipfs/go-ipld-format v0.2.0 // indirect
	github.com/ipfs/go-log/v2 v2.1.3
	github.com/kr/text v0.2.0 // indirect
	github.com/mitchellh/mapstructure v1.4.1 // indirect
	github.com/mr-tron/base58 v1.2.0 // indirect
	github.com/multiformats/go-multiaddr v0.3.3
	github.com/niemeyer/pretty v0.0.0-20200227124842-a10e7caefd8e // indirect
	github.com/nxadm/tail v1.4.8 // indirect
	github.com/pelletier/go-toml v1.6.0 // indirect
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/afero v1.2.2 // indirect
	github.com/spf13/cast v1.3.1 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.7.1 // indirect
	github.com/urfave/cli/v2 v2.3.0
	github.com/warpfork/go-wish v0.0.0-20200122115046-b9ea61034e4a // indirect
	github.com/whyrusleeping/cbor-gen v0.0.0-20210219115102-f37d292932f2 // indirect
	go.opencensus.io v0.23.0
	golang.org/x/text v0.3.5 // indirect
	golang.org/x/time v0.0.0-20200416051211-89c76fbcd5d1 // indirect
	golang.org/x/tools v0.1.1 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1
	gopkg.in/check.v1 v1.0.0-20200227125254-8fa46927fb4f // indirect
	honnef.co/go/tools v0.1.3 // indirect
)

replace github.com/filecoin-project/go-jsonrpc => github.com/ipfs-force-community/go-jsonrpc v0.1.4-0.20210521084414-5a2e6709d9bd

replace github.com/ipfs/go-ipfs-cmds => github.com/ipfs-force-community/go-ipfs-cmds v0.6.1-0.20210521090123-4587df7fa0ab
