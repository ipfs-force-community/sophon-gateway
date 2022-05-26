package proofevent

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/venus-auth/cmd/jwtclient"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	types2 "github.com/filecoin-project/venus/venus-shared/types"
	types "github.com/filecoin-project/venus/venus-shared/types/gateway"
	gtypes "github.com/ipfs-force-community/venus-gateway/types"
	"github.com/ipfs-force-community/venus-gateway/validator"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestListenProofEvent(t *testing.T) {
	addrGetter := address.NewForTestGetter()
	addr1 := addrGetter()
	addr2 := addrGetter()

	t.Run("correct", func(t *testing.T) {
		proof := setupProofEvent(t, []address.Address{addr1})
		ctx, cancel := context.WithCancel(context.Background())
		ctx = jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1")
		requestCh, err := proof.ListenProofEvent(ctx, &types.ProofRegisterPolicy{
			MinerAddress: addr1,
		})
		require.NoError(t, err)

		initReq := <-requestCh
		require.Equal(t, "InitConnect", initReq.Method)
		initBody := &types.ConnectedCompleted{}
		err = json.Unmarshal(initReq.Payload, initBody)
		require.NoError(t, err)
		channel, err := proof.getChannels(addr1)
		require.NoError(t, err)
		require.Equal(t, len(channel), 1)
		require.Equal(t, channel[0].ChannelId, initBody.ChannelId)

		//cancel and got a close request channel
		cancel()
		select {
		case <-time.After(time.Second * 30):
			t.Errorf("unable to wait for closed channel within 30s")
		case _, ok := <-requestCh:
			if !ok {
				return
			}
		}
	})

	t.Run("invalidate address", func(t *testing.T) {
		proof := setupProofEvent(t, []address.Address{addr1})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx = jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1")

		_, err := proof.ListenProofEvent(ctx, &types.ProofRegisterPolicy{
			MinerAddress: addr2,
		})
		require.Contains(t, err.Error(), "verify miner:")
	})

	t.Run("no ip exit", func(t *testing.T) {
		proof := setupProofEvent(t, []address.Address{addr1})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_, err := proof.ListenProofEvent(ctx, &types.ProofRegisterPolicy{
			MinerAddress: addr2,
		})
		require.Contains(t, err.Error(), "ip not exist")
	})
}

func TestComputeProofEvent(t *testing.T) {
	addrGetter := address.NewForTestGetter()
	addr := addrGetter()
	t.Run("correct", func(t *testing.T) {
		proof := setupProofEvent(t, []address.Address{addr})
		{
			ctx := jwtclient.CtxWithTokenLocation(context.Background(), "127.1.1.1")
			requestCh, err := proof.ListenProofEvent(ctx, &types.ProofRegisterPolicy{
				MinerAddress: addr,
			})
			require.NoError(t, err)
			go func() {
				for req := range requestCh {
					if req.Method == "ComputeProof" {
						var requestBody types.ComputeProofRequest
						err = json.Unmarshal(req.Payload, &requestBody)
						require.NoError(t, err)
						require.Equal(t, abi.PoStRandomness([]byte{1, 2, 3}), requestBody.Rand)
						require.Equal(t, abi.ChainEpoch(100), requestBody.Height)
						require.Equal(t, network.Version4, requestBody.NWVersion)

						postProof := builtin.PoStProof{
							PoStProof:  abi.RegisteredPoStProof_StackedDrgWindow8MiBV1,
							ProofBytes: []byte{1, 2, 3, 4},
						}
						result, err := json.Marshal([]builtin.PoStProof{postProof})
						require.NoError(t, err)
						err = proof.ResponseEvent(ctx, &types.ResponseEvent{
							ID:      req.ID,
							Payload: result,
							Error:   "",
						})
						require.NoError(t, err)
						return
					}
				}
			}()
		}

		{
			ctx := context.Background()
			result, err := proof.ComputeProof(ctx, addr,
				[]builtin.ExtendedSectorInfo{},
				[]byte{1, 2, 3},
				abi.ChainEpoch(100), network.Version4)
			require.NoError(t, err)
			require.Len(t, result, 1)
			require.Equal(t, abi.RegisteredPoStProof_StackedDrgWindow8MiBV1, result[0].PoStProof)
			require.Equal(t, []byte{1, 2, 3, 4}, result[0].ProofBytes)
		}
	})

	t.Run("response error", func(t *testing.T) {
		proof := setupProofEvent(t, []address.Address{addr})
		{
			ctx := jwtclient.CtxWithTokenLocation(context.Background(), "127.1.1.1")
			requestCh, err := proof.ListenProofEvent(ctx, &types.ProofRegisterPolicy{
				MinerAddress: addr,
			})
			require.NoError(t, err)
			go func() {
				for req := range requestCh {
					if req.Method == "ComputeProof" {
						err := proof.ResponseEvent(ctx, &types.ResponseEvent{
							ID:      req.ID,
							Payload: nil,
							Error:   "mock error",
						})
						require.NoError(t, err)
						return
					}
				}
			}()
		}

		{
			ctx := context.Background()
			_, err := proof.ComputeProof(ctx, addr,
				[]builtin.ExtendedSectorInfo{},
				[]byte{1, 2, 3},
				abi.ChainEpoch(100), network.Version4)
			require.EqualError(t, err, "mock error")
		}
	})

	t.Run("uncorrect result  error", func(t *testing.T) {
		proof := setupProofEvent(t, []address.Address{addr})
		{
			ctx := jwtclient.CtxWithTokenLocation(context.Background(), "127.1.1.1")
			requestCh, err := proof.ListenProofEvent(ctx, &types.ProofRegisterPolicy{
				MinerAddress: addr,
			})
			require.NoError(t, err)
			go func() {
				for req := range requestCh {
					if req.Method == "ComputeProof" {
						err := proof.ResponseEvent(ctx, &types.ResponseEvent{
							ID:      req.ID,
							Payload: []byte{1, 2, 3, 4},
							Error:   "",
						})
						require.NoError(t, err)
						return
					}
				}
			}()
		}

		{
			ctx := context.Background()
			_, err := proof.ComputeProof(ctx, addr,
				[]builtin.ExtendedSectorInfo{},
				[]byte{1, 2, 3},
				abi.ChainEpoch(100), network.Version4)
			require.Contains(t, err.Error(), "invalid character")
		}
	})

	t.Run("mistake request id error", func(t *testing.T) {
		proof := setupProofEvent(t, []address.Address{addr})
		ctx := jwtclient.CtxWithTokenLocation(context.Background(), "127.1.1.1")
		_, err := proof.ListenProofEvent(ctx, &types.ProofRegisterPolicy{
			MinerAddress: addr,
		})
		require.NoError(t, err)
		uid := types2.NewUUID()
		err = proof.ResponseEvent(ctx, &types.ResponseEvent{
			ID:      uid,
			Payload: nil,
			Error:   "",
		})

		require.EqualError(t, err, fmt.Sprintf("request id %s not exit", uid))
	})
}

func TestListConnectedMiners(t *testing.T) {
	addrGetter := address.NewForTestGetter()
	addr1 := addrGetter()
	addr2 := addrGetter()
	proof := setupProofEvent(t, []address.Address{addr1, addr2})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	{
		ctx = jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1")
		_, err := proof.ListenProofEvent(ctx, &types.ProofRegisterPolicy{
			MinerAddress: addr1,
		})
		require.NoError(t, err)
	}

	{
		ctx = jwtclient.CtxWithTokenLocation(ctx, "127.1.1.2")
		_, err := proof.ListenProofEvent(ctx, &types.ProofRegisterPolicy{
			MinerAddress: addr2,
		})
		require.NoError(t, err)
	}
	connetions, err := proof.ListConnectedMiners(ctx)
	require.NoError(t, err)
	require.Len(t, connetions, 2)
}

func TestListMinerConnection(t *testing.T) {
	addrGetter := address.NewForTestGetter()
	addr1 := addrGetter()
	proof := setupProofEvent(t, []address.Address{addr1})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	{
		ctx = jwtclient.CtxWithTokenLocation(ctx, "127.1.1.1")
		_, err := proof.ListenProofEvent(ctx, &types.ProofRegisterPolicy{
			MinerAddress: addr1,
		})
		require.NoError(t, err)
	}

	{
		ctx = jwtclient.CtxWithTokenLocation(ctx, "127.1.1.2")
		_, err := proof.ListenProofEvent(ctx, &types.ProofRegisterPolicy{
			MinerAddress: addr1,
		})
		require.NoError(t, err)
	}

	connetions, err := proof.ListMinerConnection(ctx, addr1)
	require.NoError(t, err)
	require.Equal(t, connetions.ConnectionCount, 2)
}

func setupProofEvent(t *testing.T, validateAddr []address.Address) *ProofEventStream {
	return NewProofEventStream(context.Background(), &validator.MockAuthMinerValidator{ValidatedAddr: validateAddr}, gtypes.DefaultConfig())
}
