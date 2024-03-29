package metrics

import (
	"context"

	"github.com/ipfs-force-community/metrics"
	logging "github.com/ipfs/go-log/v2"

	v2API "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"
)

var log = logging.Logger("metrics")

func SetupMetrics(ctx context.Context, metricsConfig *metrics.MetricsConfig, api v2API.IGateway) error {
	err := metrics.SetupMetrics(ctx, metricsConfig)
	if err != nil {
		return err
	}
	go recordMetricsLoop(ctx, api)
	return nil
}
