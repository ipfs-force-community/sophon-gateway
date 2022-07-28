package metrics

import (
	"context"
	"time"

	"github.com/filecoin-project/go-address"
	v1API "github.com/filecoin-project/venus/venus-shared/api/gateway/v1"
	"go.opencensus.io/stats"
	"go.opencensus.io/tag"
)

func recordMetricsLoop(ctx context.Context, api v1API.IGateway) {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			recordWalletConnectionInfo(ctx, api)
			recordMarketConnectionInfo(ctx, api)
			recordMinerConnectionInfo(ctx, api)
		case <-ctx.Done():
			log.Infof("context done, stop record metrics")
			return
		}
	}
}

func recordWalletConnectionInfo(ctx context.Context, api v1API.IGateway) {
	walletDetails, err := api.ListWalletInfo(ctx)
	if err != nil {
		log.Warnf("failed to list wallet info %v", err)
		return
	}

	addrs := make(map[address.Address]struct{})
	for _, detail := range walletDetails {
		ctx, _ = tag.New(ctx, tag.Upsert(WalletAccountKey, detail.Account))
		stats.Record(ctx, WalletNum.M(1))

		for _, conn := range detail.ConnectStates {
			_ = stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(IPKey, conn.IP)}, WalletConnNum.M(1))
			for _, addr := range conn.Addrs {
				if _, ok := addrs[addr]; ok {
					continue
				}
				addrs[addr] = struct{}{}
				_ = stats.RecordWithTags(ctx,
					[]tag.Mutator{tag.Upsert(WalletAddressKey, addr.String())}, WalletAddressNum.M(1))
			}
		}
		addrs = make(map[address.Address]struct{})
	}
}

func recordMarketConnectionInfo(ctx context.Context, api v1API.IGateway) {
	connsState, err := api.ListMarketConnectionsState(ctx)
	if err != nil {
		log.Warnf("failed to get market connections state %v", err)
		return
	}

	for _, state := range connsState {
		ctx, _ = tag.New(ctx, tag.Upsert(MinerAddressKey, state.Addr.String()), tag.Upsert(MinerTypeKey, "market"))
		for _, conn := range state.Conn.Connections {
			_ = stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(IPKey, conn.IP)}, MinerConnNum.M(1))
		}
		stats.Record(ctx, MinerNum.M(1))
	}
}

func recordMinerConnectionInfo(ctx context.Context, api v1API.IGateway) {
	miners, err := api.ListConnectedMiners(ctx)
	if err != nil {
		log.Warnf("faield to list connected miners %v", err)
		return
	}

	for _, miner := range miners {
		state, err := api.ListMinerConnection(ctx, miner)
		if err != nil {
			log.Warnf("failed to list miner connection %v", err)
			return
		}

		ctx, _ = tag.New(ctx, tag.Upsert(MinerTypeKey, "pprof"), tag.Upsert(MinerAddressKey, miner.String()))
		for _, conn := range state.Connections {
			_ = stats.RecordWithTags(ctx, []tag.Mutator{tag.Upsert(IPKey, conn.IP)}, MinerConnNum.M(1))
		}
		stats.Record(ctx, MinerNum.M(1))
	}
}
