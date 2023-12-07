package metrics

import (
	"time"

	rpcMetrics "github.com/filecoin-project/go-jsonrpc/metrics"
	"github.com/ipfs-force-community/metrics"
	"go.opencensus.io/stats"
	"go.opencensus.io/stats/view"
	"go.opencensus.io/tag"
)

// Global Tags
var (
	WalletAccountKey, _ = tag.NewKey("wallet_account")
	WalletAddressKey, _ = tag.NewKey("address")

	MinerAddressKey, _ = tag.NewKey("miner")

	MinerTypeKey, _ = tag.NewKey("miner_type")

	IPKey, _ = tag.NewKey("ip")
)

// Distribution
var defaultMillisecondsDistribution = view.Distribution(0.01, 0.05, 0.1, 0.3, 0.6, 0.8, 1, 2, 3, 4, 5, 6, 8, 10, 13, 16, 20, 25, 30, 40, 50, 65, 80, 100, 130, 160, 200, 250, 300, 400, 500, 650, 800, 1000, 2000, 3000, 4000, 5000, 7500, 10000, 20000, 50000, 100000)

var (
	// wallet
	WalletNum        = metrics.NewInt64("wallet/num", "Wallet count", stats.UnitDimensionless)
	WalletAddressNum = metrics.NewInt64("wallet/address_num", "Address owned by wallet", stats.UnitDimensionless)
	WalletConnNum    = metrics.NewInt64("wallet/conn_num", "Wallet connection count", stats.UnitDimensionless)
	WalletRegister   = stats.Int64("wallet/register", "Wallet register", stats.UnitDimensionless)
	WalletUnregister = stats.Int64("wallet/unregister", "Wallet unregister", stats.UnitDimensionless)
	WalletAddAddr    = stats.Int64("wallet/add_addr", "Wallet add a new address", stats.UnitDimensionless)
	WalletRemoveAddr = stats.Int64("wallet/remove_addr", "Wallet remove a new address", stats.UnitDimensionless)

	// miner
	MinerRegister   = metrics.NewCounter("miner/register", "Miner register", MinerAddressKey, IPKey, MinerTypeKey)
	MinerUnregister = metrics.NewCounter("miner/unregister", "Miner unregister", MinerAddressKey, IPKey, MinerTypeKey)
	MinerSource     = metrics.NewCounter("miner/source", "Miner IP", MinerAddressKey, MinerTypeKey)
	MinerNum        = metrics.NewInt64("miner/num", "Wallet count", "", MinerTypeKey)
	MinerConnNum    = metrics.NewInt64("miner/conn_num", "Miner connection count", "", MinerTypeKey)

	// method call
	WalletSign         = stats.Float64("wallet_sign", "Call WalletSign spent time", stats.UnitMilliseconds)
	WalletList         = stats.Float64("wallet_list", "Call WalletList spent time", stats.UnitMilliseconds)
	ComputeProof       = stats.Float64("compute_proof", "Call ComputeProof spent time", stats.UnitMilliseconds)
	SectorsUnsealPiece = stats.Float64("sectors_unseal_piece", "Call SectorsUnsealPiece spent time", stats.UnitMilliseconds)

	ApiState = metrics.NewInt64("api/state", "api service state. 0: down, 1: up", "")
)

var (
	// wallet
	walletRegisterView = &view.View{
		Measure:     WalletRegister,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{WalletAccountKey, IPKey},
	}
	walletUnregisterView = &view.View{
		Measure:     WalletUnregister,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{WalletAccountKey, IPKey},
	}

	walletAddAddrView = &view.View{
		Measure:     WalletAddAddr,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{WalletAccountKey, WalletAddressKey},
	}
	walletRemoveAddrView = &view.View{
		Measure:     WalletRemoveAddr,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{WalletAccountKey, WalletAddressKey},
	}

	// method call
	walletSignView = &view.View{
		Measure:     WalletSign,
		Aggregation: defaultMillisecondsDistribution,
		TagKeys:     []tag.Key{WalletAccountKey},
	}
	walletListView = &view.View{
		Measure:     WalletList,
		Aggregation: defaultMillisecondsDistribution,
		TagKeys:     []tag.Key{WalletAccountKey},
	}
	computeProofView = &view.View{
		Measure:     ComputeProof,
		Aggregation: defaultMillisecondsDistribution,
		TagKeys:     []tag.Key{MinerAddressKey},
	}
	sectorsUnsealPieceView = &view.View{
		Measure:     SectorsUnsealPiece,
		Aggregation: defaultMillisecondsDistribution,
		TagKeys:     []tag.Key{MinerAddressKey},
	}
)

var views = append([]*view.View{
	walletRegisterView,
	walletUnregisterView,
	walletAddAddrView,
	walletRemoveAddrView,
	walletSignView,
	walletListView,
	computeProofView,
	sectorsUnsealPieceView,
}, rpcMetrics.DefaultViews...)

// SinceInMilliseconds returns the duration of time since the provide time as a float64.
func SinceInMilliseconds(startTime time.Time) float64 {
	return float64(time.Since(startTime).Nanoseconds()) / 1e6
}

func init() {
	// register metrics
	_ = view.Register(views...)
}
