package types

import "github.com/filecoin-project/venus-auth/auth"

type IAuthClient interface {
	GetUser(req *auth.GetUserRequest) (*auth.OutputUser, error)
	GetMiner(req *auth.GetMinerRequest) (*auth.OutputUser, error)
	HasMiner(req *auth.HasMinerRequest) (bool, error)
}
