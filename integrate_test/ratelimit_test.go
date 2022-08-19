package integrate

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/ipfs-force-community/metrics"
	"github.com/ipfs-force-community/venus-gateway/config"
	"github.com/stretchr/testify/require"
)

func TestRateLimit(t *testing.T) {
	ctx := context.Background()

	cfg := &config.Config{
		API:       &config.APIConfig{ListenAddress: "/ip4/127.0.0.1/tcp/0"},
		Auth:      &config.AuthConfig{URL: "127.0.0.1:1"},
		Metrics:   config.DefaultConfig().Metrics,
		Trace:     &metrics.TraceConfig{JaegerTracingEnabled: false},
		RateLimit: &config.RateLimitCofnig{Redis: "27.0.0.1:6379"},
	}

	_, _, err := MockMain(ctx, []address.Address{}, t.TempDir(), cfg, defaultTestConfig())
	require.NoError(t, err)
}
