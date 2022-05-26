package validator

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/pkg/errors"
)

var _ IAuthMinerValidator = (*MockAuthMinerValidator)(nil)

type MockAuthMinerValidator struct {
	ValidatedAddr []address.Address
}

func (m MockAuthMinerValidator) Validate(ctx context.Context, miner address.Address) error {
	for _, addr := range m.ValidatedAddr {
		if miner == addr {
			return nil
		}
	}
	return errors.New("not validated address")
}
