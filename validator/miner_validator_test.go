package validator

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-address"

	"github.com/filecoin-project/venus-auth/auth"
	"github.com/filecoin-project/venus-auth/jwtclient"

	"github.com/ipfs-force-community/venus-gateway/validator/mocks"
)

var testArgs = map[string]*struct {
	validOk bool
	user    *auth.OutputUser
}{
	"test_01": {true, &auth.OutputUser{
		Id:      uuid.NewString(),
		Name:    "test_01",
		Comment: "test_01",
		State:   1,
		Miners:  []*auth.OutputMiner{{Miner: "f01001", User: "test_01"}},
	}},
	// test_02, State is disabled, so it should be invalid.
	"test_02": {false, &auth.OutputUser{
		Id:      uuid.NewString(),
		Name:    "test_02",
		Comment: "test_02",
		State:   0,
		Miners:  []*auth.OutputMiner{{Miner: "f01002", User: "test_02"}},
	}},
}

func TestAuthMinerValidator_Validate(t *testing.T) {
	address.CurrentNetwork = address.Mainnet
	authClient := mocks.NewMockAuthClient()
	validator := NewMinerValidator(authClient)

	notExistsMiner, err := address.NewIDAddress(10245566778899)
	require.NoError(t, err)

	for _, arg := range testArgs {
		authClient.AddMockUser(arg.user)
		ctx := jwtclient.CtxWithName(context.TODO(), arg.user.Name)
		for _, miner := range arg.user.Miners {
			addr, err := address.NewFromString(miner.Miner)
			require.NoError(t, err)
			if arg.validOk {
				require.NoError(t, validator.Validate(ctx, addr))
			} else {
				require.Error(t, validator.Validate(ctx, addr))
			}
		}

		require.Error(t, validator.Validate(ctx, notExistsMiner))
	}

}
