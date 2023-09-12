package cluster

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateNode(t *testing.T) {
	t.Run("test create with default port", func(t *testing.T) {
		n, err := NewNode("api1", ":7946")
		require.NoError(t, err)
		require.NotNil(t, n)
		require.Equal(t, 7946, int(n.memberShip.LocalNode().Port))
	})

	t.Run("test create with rand port", func(t *testing.T) {
		n, err := NewNode("api2", ":0")
		require.NoError(t, err)
		require.NotNil(t, n)
		require.NotEqual(t, 0, int(n.memberShip.LocalNode().Port))
	})
}

// TestNodeMeta conform that gateway node meta is can be obtain by other node correctly
func TestNodeMeta(t *testing.T) {
	node1, err := NewNode("api1", ":0")
	require.NoError(t, err)
	require.NotNil(t, node1)

	node2, err := NewNode("api2", ":0")
	require.NoError(t, err)
	require.NotNil(t, node2)

	node3, err := NewNode("api3", ":0")
	require.NoError(t, err)
	require.NotNil(t, node3)

	node3.Join(node1.Address())
	node3.Join(node2.Address())
	require.Equal(t, 3, node2.memberShip.NumMembers())

	for _, node := range node1.memberShip.Members() {
		api := getApiFromMeta(node.Meta)
		switch node.Name {
		case node1.memberShip.LocalNode().Name:
			require.Equal(t, "api1", api)
		case node2.memberShip.LocalNode().Name:
			require.Equal(t, "api2", api)
		case node3.memberShip.LocalNode().Name:
			require.Equal(t, "api3", api)
		default:
			require.Fail(t, "unexpected node name")
		}
	}
}

func TestCtxKeyBroadcastPrevent(t *testing.T) {
	ctx := context.Background()

	b := PreventBroadcast(ctx)
	require.False(t, b)

	ctx = SetPreventBroadcast(ctx)
	b = PreventBroadcast(ctx)
	require.True(t, b)
}
