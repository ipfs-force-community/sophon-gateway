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

func (m *AuthClient) RegisterSigners(userName string, signers []string) error {
	_, err := m.GetUser(&auth.GetUserRequest{Name: userName})
	if err != nil {
		return err
	}

	m.lkSigner.Lock()
	defer m.lkSigner.Unlock()
	for _, signer := range signers {
		names, ok := m.signers[signer]
		if !ok {
			m.signers[signer] = []string{userName}
		} else {
			bCreate := true
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
	}

	return nil
}

func (m *AuthClient) UnregisterSigners(userName string, signers []string) error {
	_, err := m.GetUser(&auth.GetUserRequest{Name: userName})
	if err != nil {
		return err
	}

	m.lkSigner.Lock()
	defer m.lkSigner.Unlock()
	for _, signer := range signers {
		names, ok := m.signers[signer]
		if ok {
			idx := 0
			for _, name := range names {
				if name != userName {
					names[idx] = name
					idx++
				}
			}
			m.signers[signer] = names[:idx]
		}
	}

	return nil
}

func (m *AuthClient) VerifyUsers(names []string) error {
	m.lkUser.Lock()
	defer m.lkUser.Unlock()

	for _, name := range names {
		if _, ok := m.users[name]; !ok {
			return errors.New("not exist")
		}
	}

	return nil
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

var (
	_ jwtclient.IAuthClient    = (*AuthClient)(nil)
	_ ratelimit.ILimitFinder   = (*AuthClient)(nil)
	_ jwtclient.IJwtAuthClient = (*AuthClient)(nil)
)
