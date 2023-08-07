package proxy

import (
	"errors"
	"net/url"
	"testing"

	chainV0 "github.com/filecoin-project/venus/venus-shared/api/chain/v0"
	"github.com/stretchr/testify/require"
)

func TestRegisterProxyHeader(t *testing.T) {
	t.Run("test invalid header", func(t *testing.T) {
		proxy := NewProxy()
		_, err := proxy.getReverseHandler("test-header")
		require.Error(t, err)
		require.True(t, errors.Is(err, ErrorInvalidHeader))
	})

	t.Run("test default header", func(t *testing.T) {
		proxy := NewProxy()
		_, err := proxy.getReverseHandler(chainV0.APINamespace)
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrorInvalidHeader)
		require.ErrorIs(t, err, ErrorNoReverseProxyRegistered)
	})

	t.Run("test custom header", func(t *testing.T) {
		Header2HostPreset["test-header"] = HostUnknown
		proxy := NewProxy()
		_, err := proxy.getReverseHandler("test-header")
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrorInvalidHeader)
		require.ErrorIs(t, err, ErrorNoReverseProxyRegistered)
	})
}

func TestRegisterReverseProxy(t *testing.T) {
	t.Run("test register reverse proxy", func(t *testing.T) {
		proxy := NewProxy()
		_, err := proxy.getReverseHandler(chainV0.APINamespace)
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrorInvalidHeader)
		require.ErrorIs(t, err, ErrorNoReverseProxyRegistered)

		u, err := url.Parse("http://localhost")
		require.NoError(t, err)

		proxy.RegisterReverseHandler(HostNode, NewReverseServer(u))
		_, err = proxy.getReverseHandler(chainV0.APINamespace)
		require.NoError(t, err)

		// unset
		proxy.RegisterReverseHandler(HostNode, nil)
		_, err = proxy.getReverseHandler(chainV0.APINamespace)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrorNoReverseProxyRegistered)

		proxy.RegisterReverseHandler(HostNode, NewReverseServer(u))
		_, err = proxy.getReverseHandler(chainV0.APINamespace)
		require.NoError(t, err)

		// unset by empty addr
		proxy.RegisterReverseByAddr(HostNode, "")
		_, err = proxy.getReverseHandler(chainV0.APINamespace)
		require.Error(t, err)
		require.ErrorIs(t, err, ErrorNoReverseProxyRegistered)
	})
}
