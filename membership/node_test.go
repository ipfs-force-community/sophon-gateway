package membership

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCreateNode(t *testing.T) {
	t.Run("test create with default port", func(t *testing.T) {
		n, err := NewNode("node1", "api1", 7946)
		require.NoError(t, err)
		require.NotNil(t, n)
		require.Equal(t, "node1", n.memberShip.LocalNode().Name)
		require.Equal(t, 7946, int(n.memberShip.LocalNode().Port))
	})

	t.Run("test create with rand port", func(t *testing.T) {
		n, err := NewNode("node_name", "api2", 0)
		require.NoError(t, err)
		require.NotNil(t, n)
		require.Equal(t, "node_name", n.memberShip.LocalNode().Name)
		require.NotEqual(t, 0, int(n.memberShip.LocalNode().Port))
	})
}

// TestNodeMeta conform that gateway node meta is can be obtain by other node correctly
func TestNodeMeta(t *testing.T) {

	node1, err := NewNode("node1", "api1", 0)
	require.NoError(t, err)
	require.NotNil(t, node1)

	node2, err := NewNode("node2", "api2", 0)
	require.NoError(t, err)
	require.NotNil(t, node2)

	node3, err := NewNode("node3", "api3", 0)
	require.NoError(t, err)
	require.NotNil(t, node3)

	node3.Join(node1.Address())
	node3.Join(node2.Address())
	require.Equal(t, 3, node2.memberShip.NumMembers())

	for _, node := range node1.memberShip.Members() {
		api := getApiFromMeta(node.Meta)
		switch node.Name {
		case "node1":
			require.Equal(t, "api1", api)
		case "node2":
			require.Equal(t, "api2", api)
		case "node3":
			require.Equal(t, "api3", api)
		default:
			require.Fail(t, "unexpected node name")
		}
	}
}
