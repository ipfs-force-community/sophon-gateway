package validator

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/venus-auth/core"
	"github.com/filecoin-project/venus-auth/jwtclient"
)

type AuthMinerValidator struct {
	authClient jwtclient.IAuthClient
}

type IAuthMinerValidator interface {
	Validate(ctx context.Context, miner address.Address) error
}

var _ IAuthMinerValidator = (*AuthMinerValidator)(nil)

func (amv *AuthMinerValidator) Validate(ctx context.Context, miner address.Address) error {
	account, exist := jwtclient.CtxGetName(ctx)
	if !exist {
		return fmt.Errorf("user name not exist in rpc context")
	}
	user, err := amv.authClient.GetUserByMiner(ctx, miner)
	if err != nil {
		return fmt.Errorf("get user by miner(%s), failed:%w", miner.String(), err)
	}
	if user.State != core.UserStateEnabled {
		return fmt.Errorf("user:%s is disabled, please enable it on 'venus-auth'", account)
	}
	if user.Name != account {
		return fmt.Errorf("your account is:%s, but miner:%s is currently bind to user:%s, change this on 'venus-auth'",
			account, miner.String(), user.Name)
	}
	return nil
}

func NewMinerValidator(authClient jwtclient.IAuthClient) IAuthMinerValidator {
	return &AuthMinerValidator{authClient: authClient}
}
