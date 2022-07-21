package integrate

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/ipfs-force-community/venus-gateway/types"
	"github.com/stretchr/testify/require"
)

func TestRateLimit(t *testing.T) {
	ctx := context.Background()

	cfg := &types.Config{
		Listen:         "/ip4/127.0.0.1/tcp/0",
		AuthUrl:        "127.0.0.1:1",
		JaegerProxy:    "",
		TraceSampler:   0,
		TraceNodeName:  "",
		RateLimitRedis: "127.0.0.1:6379",
	}
	_, _, err := MockMain(ctx, []address.Address{}, cfg, defaultTestConfig())
	require.NoError(t, err)
}
