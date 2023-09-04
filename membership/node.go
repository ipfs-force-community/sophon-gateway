package membership

import (
	"encoding/json"

	"github.com/hashicorp/memberlist"
	logging "github.com/ipfs/go-log/v2"
)

var log = logging.Logger("gateway_node")

type metaKey string

const (
	metaKeyApi metaKey = "api"
)

type Node struct {
	meta       map[metaKey]string
	memberShip *memberlist.Memberlist
}

func NewNode(name, api string, port int) (*Node, error) {
	n := &Node{
		meta: map[metaKey]string{metaKeyApi: api},
	}

	// config init
	cfg := memberlist.DefaultLocalConfig()
	cfg.Name = name
	cfg.BindPort = port
	cfg.Delegate = n
	cfg.Events = n

	list, err := memberlist.Create(cfg)
	if err != nil {
		return nil, err
	}

	n.memberShip = list
	return n, nil
}

func (n *Node) Join(addr string) error {
	_, err := n.memberShip.Join([]string{addr})
	return err
}

func (n *Node) Address() string {
	return n.memberShip.LocalNode().Address()
}

var _ memberlist.EventDelegate = (*Node)(nil)

func (n *Node) NotifyJoin(node *memberlist.Node) {
	log.Info("node join: ", node.Name)
}

func (n *Node) NotifyLeave(node *memberlist.Node) {
	log.Info("node leave: ", node.Name)
}

func (n *Node) NotifyUpdate(node *memberlist.Node) {
	// log.Info("node update: ", node.Name)
}

var _ memberlist.Delegate = (*Node)(nil)

func (n *Node) NodeMeta(limit int) []byte {
	b, err := json.Marshal(n.meta)
	if err != nil {
		log.Error("marshal node meta err: ", err)
		return nil
	}
	return b
}

func (n *Node) LocalState(join bool) []byte {
	return nil
}

func (n *Node) MergeRemoteState(buf []byte, join bool) {
}

func (n *Node) NotifyMsg([]byte) {
	// ignore all msg
}

func (n *Node) GetBroadcasts(overhead, limit int) [][]byte {
	return nil
}

func getApiFromMeta(b []byte) string {
	var meta map[metaKey]string
	if err := json.Unmarshal(b, &meta); err != nil {
		log.Error("unmarshal node meta err: ", err)
		return ""
	}

	return meta[metaKeyApi]
}
