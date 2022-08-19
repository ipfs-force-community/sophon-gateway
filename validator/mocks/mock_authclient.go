package mocks

import (
	"context"
	"errors"
	"fmt"

	"github.com/filecoin-project/go-address"
	rpcAuth "github.com/filecoin-project/go-jsonrpc/auth"

	"github.com/filecoin-project/venus-auth/auth"
	"github.com/filecoin-project/venus-auth/jwtclient"

	"github.com/ipfs-force-community/metrics/ratelimit"

	"github.com/ipfs-force-community/venus-gateway/types"
)

type AuthClient struct {
	// key: username, v: user
	users map[string]*auth.OutputUser

	// key: miner address, v: username
	miners map[string]string

	// key: signer address, v: username
	signers map[string]string
}

func (m AuthClient) GetUser(req *auth.GetUserRequest) (*auth.OutputUser, error) {
	if user, ok := m.users[req.Name]; ok {
		return user, nil
	}

	return nil, errors.New("not exist")
}

func (m *AuthClient) GetUserByMiner(req *auth.GetUserByMinerRequest) (*auth.OutputUser, error) {
	username, ok := m.miners[req.Miner]
	if !ok {
		return nil, errors.New("not exist")
	}

	if user, ok := m.users[username]; ok {
		return user, nil
	}

	return nil, errors.New("not exist")
}

func (m *AuthClient) GetUserBySigner(req *auth.GetUserBySignerRequest) (*auth.OutputUser, error) {
	username, ok := m.signers[req.Signer]
	if !ok {
		return nil, errors.New("not exist")
	}

	if user, ok := m.users[username]; ok {
		return user, nil
	}

	return nil, errors.New("not exist")
}

func (m *AuthClient) UpsertMiner(userName, miner string) (bool, error) {
	_, err := m.GetUser(&auth.GetUserRequest{Name: userName})
	if err != nil {
		return false, err
	}

	_, bUpdate := m.miners[miner]
	m.miners[miner] = userName

	// The original intention of venus-auth is to return true for creation and false for update
	return !bUpdate, nil
}

func (m AuthClient) HasMiner(req *auth.HasMinerRequest) (bool, error) {
	username, ok := m.miners[req.Miner]

	if !ok {
		return ok, nil
	}

	if len(req.User) > 0 {
		if username != req.User {
			return false, nil
		}
	}

	return ok, nil
}

func (m AuthClient) UpsertSigner(userName, signer string) (bool, error) {
	_, err := m.GetUser(&auth.GetUserRequest{Name: userName})
	if err != nil {
		return false, err
	}

	_, bUpdate := m.signers[signer]
	m.signers[signer] = userName

	// The original intention of venus-auth is to return true for creation and false for update
	return !bUpdate, nil
}

func (m AuthClient) HasSigner(req *auth.HasSignerRequest) (bool, error) {
	username, ok := m.signers[req.Signer]

	if !ok {
		return ok, nil
	}

	if len(req.User) > 0 {
		if username != req.User {
			return false, nil
		}
	}

	return ok, nil
}

func (m *AuthClient) AddMockUser(users ...*auth.OutputUser) {
	for _, user := range users {
		m.users[user.Name] = user
		for _, miner := range user.Miners {
			m.miners[miner.Miner] = miner.User
		}
	}
}

func (m *AuthClient) GetUserLimit(username, service, api string) (*ratelimit.Limit, error) {
	if _, ok := m.users[username]; !ok {
		return nil, fmt.Errorf("%s not exist", username)
	}

	return &ratelimit.Limit{Account: username}, nil
}

func (m *AuthClient) Verify(ctx context.Context, token string) ([]rpcAuth.Permission, error) {
	panic("Don't call me")
}

func NewMockAuthClient() *AuthClient {
	address.CurrentNetwork = address.Mainnet
	return &AuthClient{
		users:   make(map[string]*auth.OutputUser),
		miners:  make(map[string]string),
		signers: make(map[string]string),
	}
}

var _ types.IAuthClient = (*AuthClient)(nil)
var _ ratelimit.ILimitFinder = (*AuthClient)(nil)
var _ jwtclient.IJwtAuthClient = (*AuthClient)(nil)
