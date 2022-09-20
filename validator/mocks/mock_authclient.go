package mocks

import (
	"context"
	"errors"
	"fmt"
	"sync"

	rpcAuth "github.com/filecoin-project/go-jsonrpc/auth"

	"github.com/filecoin-project/venus-auth/auth"
	"github.com/filecoin-project/venus-auth/jwtclient"

	"github.com/ipfs-force-community/metrics/ratelimit"

	"github.com/ipfs-force-community/venus-gateway/types"
)

type AuthClient struct {
	// key: username, v: user
	users  map[string]*auth.OutputUser
	lkUser sync.RWMutex

	// key: signer address, v: username
	signers  map[string][]string
	lkSigner sync.RWMutex
}

func (m *AuthClient) GetUser(req *auth.GetUserRequest) (*auth.OutputUser, error) {
	m.lkUser.Lock()
	defer m.lkUser.Unlock()

	if user, ok := m.users[req.Name]; ok {
		return user, nil
	}

	return nil, errors.New("not exist")
}

func (m *AuthClient) GetUserByMiner(req *auth.GetUserByMinerRequest) (*auth.OutputUser, error) {
	m.lkUser.Lock()
	defer m.lkUser.Unlock()

	for _, user := range m.users {
		for _, miner := range user.Miners {
			if req.Miner == miner.Miner {
				return user, nil
			}
		}
	}

	return nil, errors.New("not exist")
}

func (m *AuthClient) GetUserBySigner(signer string) (auth.ListUsersResponse, error) {
	m.lkSigner.Lock()
	names, ok := m.signers[signer]
	m.lkSigner.Unlock()
	if !ok {
		return nil, errors.New("not exist")
	}

	m.lkUser.Lock()
	defer m.lkUser.Unlock()
	users := make(auth.ListUsersResponse, 0)
	for _, name := range names {
		if user, ok := m.users[name]; ok {
			users = append(users, user)
		}
	}

	return users, nil
}

func (m *AuthClient) RegisterSigner(userName, signer string) (bool, error) {
	_, err := m.GetUser(&auth.GetUserRequest{Name: userName})
	if err != nil {
		return false, err
	}

	bCreate := true
	m.lkSigner.Lock()
	defer m.lkSigner.Unlock()
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

func (m *AuthClient) UnregisterSigner(userName, signer string) (bool, error) {
	_, err := m.GetUser(&auth.GetUserRequest{Name: userName})
	if err != nil {
		return false, err
	}

	bDel := false
	m.lkSigner.Lock()
	defer m.lkSigner.Unlock()
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
	m.lkUser.Lock()
	defer m.lkUser.Unlock()

	for _, user := range users {
		m.users[user.Name] = user
	}
}

func (m *AuthClient) GetUserLimit(username, service, api string) (*ratelimit.Limit, error) {
	m.lkUser.Lock()
	defer m.lkUser.Unlock()

	if _, ok := m.users[username]; !ok {
		return nil, fmt.Errorf("%s not exist", username)
	}

	return &ratelimit.Limit{Account: username}, nil
}

func (m *AuthClient) Verify(ctx context.Context, token string) ([]rpcAuth.Permission, error) {
	panic("Don't call me")
}

func NewMockAuthClient() *AuthClient {
	return &AuthClient{
		users:   make(map[string]*auth.OutputUser),
		signers: make(map[string][]string),
	}
}

var _ types.IAuthClient = (*AuthClient)(nil)
var _ ratelimit.ILimitFinder = (*AuthClient)(nil)
var _ jwtclient.IJwtAuthClient = (*AuthClient)(nil)
