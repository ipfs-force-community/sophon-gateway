package validator

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"

	"github.com/ipfs-force-community/sophon-auth/core"
	"github.com/ipfs-force-community/sophon-auth/jwtclient"
)

type AuthMinerValidator struct {
	authClient jwtclient.IAuthClient
}

type IAuthMinerValidator interface {
	Validate(ctx context.Context, miner address.Address) error
}

var _ IAuthMinerValidator = (*AuthMinerValidator)(nil)

func (amv *AuthMinerValidator) Validate(ctx context.Context, miner address.Address) error {
	account, exist := core.CtxGetName(ctx)
	if !exist {
		return fmt.Errorf("user name not exist in rpc context")
	}

	ok, err := amv.authClient.MinerExistInUser(ctx, account, miner)
	if err != nil {
		return fmt.Errorf("check miner(%s) exist in user(%s), failed:%w", miner.String(), account, err)
	}
	if !ok {
		return fmt.Errorf("miner:%s not exist in user:%s, please bind it on 'sophon-auth'", miner.String(), account)
	}

	return nil
}

func NewMinerValidator(authClient jwtclient.IAuthClient) IAuthMinerValidator {
	return &AuthMinerValidator{authClient: authClient}
}
