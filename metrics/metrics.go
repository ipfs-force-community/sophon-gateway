package metrics

import (
	"time"

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
	WalletRegister   = stats.Int64("wallet_register", "Wallet register", stats.UnitDimensionless)
	WalletUnregister = stats.Int64("wallet_unregister", "Wallet unregister", stats.UnitDimensionless)
	WalletNum        = stats.Int64("wallet_num", "Wallet count", stats.UnitDimensionless)
	WalletAddressNum = stats.Int64("wallet_address_num", "Address owned by wallet", stats.UnitDimensionless)
	WalletSource     = stats.Int64("wallet_source", "Wallet IP", stats.UnitDimensionless)
	WalletAddAddr    = stats.Int64("wallet_add_addr", "Wallet add a new address", stats.UnitDimensionless)
	WalletRemoveAddr = stats.Int64("wallet_remove_addr", "Wallet remove a new address", stats.UnitDimensionless)
	WalletConnNum    = stats.Int64("wallet_conn_num", "Wallet connection count", stats.UnitDimensionless)

	// miner
	MinerRegister   = stats.Int64("miner_register", "Miner register", stats.UnitDimensionless)
	MinerUnregister = stats.Int64("miner_unregister", "Miner unregister", stats.UnitDimensionless)
	MinerNum        = stats.Int64("miner_num", "Wallet count", stats.UnitDimensionless)
	MinerSource     = stats.Int64("wallet_source", "Miner IP", stats.UnitDimensionless)
	MinerConnNum    = stats.Int64("miner_conn_num", "Miner connection count", stats.UnitDimensionless)

	// method call
	WalletSign         = stats.Float64("wallet_sign", "Call WalletSign spent time", stats.UnitMilliseconds)
	WalletList         = stats.Float64("wallet_list", "Call WalletList spent time", stats.UnitMilliseconds)
	ComputeProof       = stats.Float64("compute_proof", "Call ComputeProof spent time", stats.UnitMilliseconds)
	IsUnsealed         = stats.Float64("is_unsealed", "Call IsUnsealed spent time", stats.UnitMilliseconds)
	SectorsUnsealPiece = stats.Float64("sectors_unseal_piece", "Call SectorsUnsealPiece spent time", stats.UnitMilliseconds)
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
	walletNumView = &view.View{
		Measure:     WalletNum,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{WalletAccountKey},
	}
	walletAddressNumView = &view.View{
		Measure:     WalletAddressNum,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{WalletAccountKey, WalletAddressKey},
	}
	walletSourceView = &view.View{
		Measure:     WalletSource,
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
	walletConnNumView = &view.View{
		Measure:     WalletConnNum,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{WalletAccountKey, IPKey},
	}

	// miner
	minerRegisterView = &view.View{
		Measure:     MinerRegister,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{MinerAddressKey, MinerTypeKey, IPKey},
	}
	minerUnregisterView = &view.View{
		Measure:     MinerUnregister,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{MinerAddressKey, MinerTypeKey, IPKey},
	}
	minerNumView = &view.View{
		Measure:     MinerNum,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{MinerAddressKey, MinerTypeKey},
	}
	minerSourceView = &view.View{
		Measure:     MinerSource,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{MinerAddressKey, MinerTypeKey},
	}
	minerConnNumView = &view.View{
		Measure:     WalletConnNum,
		Aggregation: view.Count(),
		TagKeys:     []tag.Key{MinerAddressKey, IPKey, MinerTypeKey},
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
	isUnsealedView = &view.View{
		Measure:     IsUnsealed,
		Aggregation: defaultMillisecondsDistribution,
		TagKeys:     []tag.Key{MinerAddressKey},
	}
	sectorsUnsealPieceView = &view.View{
		Measure:     SectorsUnsealPiece,
		Aggregation: defaultMillisecondsDistribution,
		TagKeys:     []tag.Key{MinerAddressKey},
	}
)

var views = []*view.View{
	walletRegisterView,
	walletUnregisterView,
	walletNumView,
	walletAddressNumView,
	walletSourceView,
	walletAddAddrView,
	walletRemoveAddrView,
	walletConnNumView,

	minerRegisterView,
	minerUnregisterView,
	minerNumView,
	minerSourceView,
	minerConnNumView,

	walletSignView,
	walletListView,
	computeProofView,
	isUnsealedView,
	sectorsUnsealPieceView,
}

// SinceInMilliseconds returns the duration of time since the provide time as a float64.
func SinceInMilliseconds(startTime time.Time) float64 {
	return float64(time.Since(startTime).Nanoseconds()) / 1e6
}
