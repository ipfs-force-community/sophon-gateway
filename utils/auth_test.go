package utils

import (
	"context"
	"testing"

	"github.com/filecoin-project/go-jsonrpc/auth"

	"github.com/stretchr/testify/require"
)

func TestLocalJwtCreateAndVerify(t *testing.T) {
	ctx := context.Background()
	jwt, err := NewLocalJwtClient(t.TempDir())
	require.NoError(t, err)
	perm, err := jwt.Verify(ctx, string(jwt.Token))
	require.NoError(t, err)
	require.Equal(t, []auth.Permission{"admin", "sign", "write", "read"}, perm)
}
