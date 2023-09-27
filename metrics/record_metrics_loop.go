package metrics

import (
	"context"
	"time"

	"github.com/filecoin-project/go-address"
	"go.opencensus.io/tag"

	v2API "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"
)

func recordMetricsLoop(ctx context.Context, api v2API.IGateway) {
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

func recordWalletConnectionInfo(ctx context.Context, api v2API.IGateway) {
	walletDetails, err := api.ListWalletInfo(ctx)
	if err != nil {
		log.Warnf("failed to list wallet info %v", err)
		return
	}

	var walletNum, connNum int64
	addrs := make(map[address.Address]struct{})
	for _, detail := range walletDetails {
		ctx, _ = tag.New(ctx, tag.Upsert(WalletAccountKey, detail.Account))
		walletNum++

		for _, conn := range detail.ConnectStates {
			connNum++
			for _, addr := range conn.Addrs {
				if _, ok := addrs[addr]; ok {
					continue
				}
				addrs[addr] = struct{}{}
			}
		}
	}

	WalletNum.Set(ctx, walletNum)
	WalletConnNum.Set(ctx, connNum)
	WalletAddressNum.Set(ctx, int64(len(addrs)))
}

func recordMarketConnectionInfo(ctx context.Context, api v2API.IGateway) {
	ctx, _ = tag.New(ctx, tag.Upsert(MinerTypeKey, "market"))
	connsState, err := api.ListMarketConnectionsState(ctx)
	if err != nil {
		log.Warnf("failed to get market connections state %v", err)
		return
	}

	var connNum int64
	for _, state := range connsState {

		connNum += int64(len(state.Conn.Connections))
	}
	MinerConnNum.Set(ctx, connNum)
	MinerNum.Set(ctx, int64(len(connsState)))
}

func recordMinerConnectionInfo(ctx context.Context, api v2API.IGateway) {
	ctx, _ = tag.New(ctx, tag.Upsert(MinerTypeKey, "pprof"))

	miners, err := api.ListConnectedMiners(ctx)
	if err != nil {
		log.Warnf("faield to list connected miners %v", err)
		return
	}

	var connNum int64
	for _, miner := range miners {
		state, err := api.ListMinerConnection(ctx, miner)
		if err != nil {
			log.Warnf("failed to list miner connection %v", err)
			return
		}

		connNum += int64(len(state.Connections))
	}
	MinerConnNum.Set(ctx, connNum)
	MinerNum.Set(ctx, int64(len(miners)))
}
