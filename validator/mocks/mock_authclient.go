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
	signers map[string][]string
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

func (m *AuthClient) GetUserBySigner(signer string) (auth.ListUsersResponse, error) {
	names, ok := m.signers[signer]
	if !ok {
		return nil, errors.New("not exist")
	}

	users := make(auth.ListUsersResponse, 0)
	for _, name := range names {
		if user, ok := m.users[name]; ok {
			users = append(users, user)
		}
	}

	return users, nil
}

func (m AuthClient) RegisterSigner(userName, signer string) (bool, error) {
	_, err := m.GetUser(&auth.GetUserRequest{Name: userName})
	if err != nil {
		return false, err
	}

	bCreate := true
	names, ok := m.signers[signer]
	if !ok {
		m.signers[signer] = []string{userName}
	} else {
		for _, name := range names {
			if name == userName {
				bCreate = false
				break
			}
		}

		if bCreate {
			names = append(names, userName)
			m.signers[signer] = names
		}
	}

	// The original intention of venus-auth is to return true for creation and false for update
	return bCreate, nil
}

func (m AuthClient) UnregisterSigner(userName, signer string) (bool, error) {
	bDel := false

	_, err := m.GetUser(&auth.GetUserRequest{Name: userName})
	if err != nil {
		return false, err
	}

	names, ok := m.signers[signer]
	if ok {
		idx := 0
		for _, name := range names {
			if name != userName {
				names[idx] = name
				idx++
			} else {
				bDel = true
			}
		}
		m.signers[signer] = names[:idx]
	}

	return bDel, nil
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
		signers: make(map[string][]string),
	}
}

var _ types.IAuthClient = (*AuthClient)(nil)
var _ ratelimit.ILimitFinder = (*AuthClient)(nil)
var _ jwtclient.IJwtAuthClient = (*AuthClient)(nil)
