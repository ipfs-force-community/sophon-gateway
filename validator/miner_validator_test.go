// stm: #unit
package validator

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/venus/venus-shared/types"

	"github.com/ipfs-force-community/sophon-auth/auth"
	"github.com/ipfs-force-community/sophon-auth/core"

	"github.com/ipfs-force-community/sophon-gateway/validator/mocks"
)

var (
	f01001 = types.MustParseAddress("f01001")
	f01002 = types.MustParseAddress("f01002")
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
		Miners:  []*auth.OutputMiner{{Miner: f01001, User: "test_01"}},
	}},
	// test_02, State is disabled, so it should be invalid.
	"test_02": {false, &auth.OutputUser{
		Id:      uuid.NewString(),
		Name:    "test_02",
		Comment: "test_02",
		State:   0,
		Miners:  []*auth.OutputMiner{{Miner: f01002, User: "test_02"}},
	}},
	// test_03, username is not same as miner
	"test_03": {false, &auth.OutputUser{
		Id:      uuid.NewString(),
		Name:    "test_02",
		Comment: "test_02",
		State:   1,
		Miners:  []*auth.OutputMiner{{Miner: f01002, User: "test_02"}},
	}},
	// username not exists in rpc context
	"": {false, &auth.OutputUser{
		Id:      uuid.NewString(),
		Name:    "test_02",
		Comment: "test_02",
		State:   1,
		Miners:  []*auth.OutputMiner{{Miner: f01002, User: "test_02"}},
	}},
}

func TestAuthMinerValidator_Validate(t *testing.T) {
	ctx := context.TODO()
	address.CurrentNetwork = address.Mainnet
	authClient := mocks.NewMockAuthClient()
	validator := NewMinerValidator(authClient)

	notExistsMiner, err := address.NewIDAddress(10245566778899)
	require.NoError(t, err)

	for userName, arg := range testArgs {
		authClient.AddMockUser(ctx, arg.user)
		var ctx = context.Background()
		if userName != "" {
			ctx = core.CtxWithName(ctx, userName)
		}

		// miner not exists
		// stm: @VENUSGATEWAY_VALIDATOR_VALIDATE_004
		require.Error(t, validator.Validate(ctx, notExistsMiner))
	}
}
