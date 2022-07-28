package utils

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"io/ioutil"
	"path"

	"github.com/filecoin-project/go-jsonrpc/auth"
	auth2 "github.com/filecoin-project/venus-auth/auth"
	"github.com/filecoin-project/venus-auth/cmd/jwtclient"
	"github.com/filecoin-project/venus-auth/core"
	jwt3 "github.com/gbrlsnchs/jwt/v3"
)

const TokenFile = "token"

// todo: this is a temporary solution
type LocalJwtClient struct {
	repo   string
	Seckey []byte
	Token  []byte
}

func NewLocalJwtClient(repo string) (*LocalJwtClient, error) {
	var err error
	var seckey []byte
	if seckey, err = ioutil.ReadAll(io.LimitReader(rand.Reader, 32)); err != nil {
		return nil, err
	}
	var cliToken []byte
	if cliToken, err = jwt3.Sign(
		auth2.JWTPayload{
			Perm: core.PermAdmin,
			Name: "GateWayLocalToken",
		}, jwt3.NewHS256(seckey)); err != nil {
		return nil, err
	}

	return &LocalJwtClient{
		repo:   repo,
		Seckey: seckey,
		Token:  cliToken,
	}, nil
}

func (l *LocalJwtClient) Verify(ctx context.Context, token string) ([]auth.Permission, error) {
	var payload auth2.JWTPayload
	if _, err := jwt3.Verify([]byte(token), jwt3.NewHS256(l.Seckey), &payload); err != nil {
		return nil, fmt.Errorf("JWT Verification failed: %v", err)
	}
	jwtPerms := core.AdaptOldStrategy(payload.Perm)
	perms := make([]auth.Permission, len(jwtPerms))
	copy(perms, jwtPerms)
	return perms, nil
}

func (l *LocalJwtClient) SaveToken() error {
	return ioutil.WriteFile(path.Join(l.repo, TokenFile), l.Token, 0644)
}

var _ jwtclient.IJwtAuthClient = (*LocalJwtClient)(nil)
