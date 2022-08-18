package types

import (
	"github.com/filecoin-project/venus-auth/auth"
)

type IAuthClient interface {
	GetUser(req *auth.GetUserRequest) (*auth.OutputUser, error)
	GetUserByMiner(req *auth.GetUserByMinerRequest) (*auth.OutputUser, error)
	GetUserBySigner(req *auth.GetUserBySignerRequest) (*auth.OutputUser, error)
	UpsertMiner(user, miner string) (bool, error)
	HasMiner(req *auth.HasMinerRequest) (bool, error)
	UpsertSigner(user, addr string) (bool, error)
	HasSigner(req *auth.HasSignerRequest) (bool, error)
}
