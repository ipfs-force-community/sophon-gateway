package main

import (
	"context"
	"fmt"
	"github.com/filecoin-project/go-jsonrpc/auth"
	"github.com/filecoin-project/venus-auth/cmd/jwtclient"
	"github.com/ipfs-force-community/venus-gateway/cmds"
)

// todo: this is a temporary solution
type localJwtClient struct{}

var AllPermisstion = []auth.Permission{"read", "write", "sign", "admin"}

func (l localJwtClient) Verify(ctx context.Context, token string) ([]auth.Permission, error) {
	if token == cmds.VenusGateWayLocalToken {
		return AllPermisstion, nil
	}
	return []auth.Permission{}, fmt.Errorf("local JwtVerification failed")
}

var _ jwtclient.IJwtAuthClient = (*localJwtClient)(nil)
