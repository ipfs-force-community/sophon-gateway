package types

import (
	"github.com/filecoin-project/venus-auth/auth"
)

type IAuthClient interface {
	GetUser(req *auth.GetUserRequest) (*auth.OutputUser, error)
	GetUserByMiner(req *auth.GetUserByMinerRequest) (*auth.OutputUser, error)
	GetUserBySigner(signer string) (auth.ListUsersResponse, error)
	RegisterSigner(user, addr string) (bool, error)
	UnregisterSigner(user, addr string) (bool, error)
}
