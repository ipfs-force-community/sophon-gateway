package mocks

import (
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/venus-auth/auth"
	"github.com/ipfs-force-community/venus-gateway/types"
	"golang.org/x/xerrors"
)

type AuthClient struct {
	//users  map[string]*auth.OutputUser // key: username, v: user
	miners map[string]*auth.OutputUser // key: miner, v: user
}

func (m AuthClient) GetUser(req *auth.GetUserRequest) (*auth.OutputUser, error) {
	panic("implement me")
}

func (m *AuthClient) GetUserByMiner(req *auth.GetUserByMinerRequest) (*auth.OutputUser, error) {
	if len(m.miners) == 0 {
		return nil, xerrors.Errorf("not exists")
	}
	user, exists := m.miners[req.Miner]
	if !exists {
		return nil, xerrors.Errorf("not exists")
	}
	return user, nil
}

func (m AuthClient) HasMiner(req *auth.HasMinerRequest) (bool, error) {
	panic("implement me")
}

func (m *AuthClient) AddMockUser(users ...*auth.OutputUser) {
	//m.users[user.Name] = user
	for _, user := range users {
		for _, miner := range user.Miners {
			m.miners[miner.Miner] = user
		}
	}
}

func NewMockAuthClient() *AuthClient {
	address.CurrentNetwork = address.Mainnet
	return &AuthClient{miners: make(map[string]*auth.OutputUser)}
}

var _ types.IAuthClient = (*AuthClient)(nil)
