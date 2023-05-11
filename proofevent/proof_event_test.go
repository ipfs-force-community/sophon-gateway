// stm: #unit
package proofevent

import (
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/venus-gateway/testhelper"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/venus-auth/core"
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

	t.Run("init connect", func(t *testing.T) {
		proof := setupProofEvent(t, []address.Address{addr1})
		ctx, cancel := context.WithCancel(context.Background())
		ctx = core.CtxWithTokenLocation(ctx, "127.1.1.1")
		// stm: @VENUSGATEWAY_PROOF_EVENT_LISTEN_PROOF_EVENT_001
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

		// cancel and got a close request channel
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

	t.Run("data race", func(t *testing.T) {
		proof := setupProofEvent(t, []address.Address{addr1})
		ctx, cancel := context.WithCancel(context.Background())
		proofClient := NewProofEvent(proof, addr1, testhelper.NewTimeoutProofHandler(time.Second), log.With())
		// stm:
		go proofClient.ListenProofRequest(core.CtxWithTokenLocation(ctx, "127.1.1.1"))
		proofClient.WaitReady(ctx)

		// fill request
		wg := sync.WaitGroup{}
		reqCount := gtypes.DefaultConfig().RequestQueueSize * 2
		wg.Add(reqCount)
		for i := 0; i < reqCount; i++ {
			go func() {
				wg.Done()
				_, _ = proof.ComputeProof(context.Background(), addr1, []builtin.ExtendedSectorInfo{}, []byte{}, 100, 16)
			}()
		}
		wg.Wait()

		//cancel and got a close request channel
		cancel()

		wg2 := sync.WaitGroup{}
		wg2.Add(reqCount)
		go func() {
			for i := 0; i < reqCount; i++ {
				defer wg2.Done()
				_, _ = proof.ComputeProof(context.Background(), addr1, []builtin.ExtendedSectorInfo{}, []byte{}, 100, 16)
			}
		}()
		wg2.Wait()
	})

	t.Run("invalidate address", func(t *testing.T) {
		proof := setupProofEvent(t, []address.Address{addr1})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		ctx = core.CtxWithTokenLocation(ctx, "127.1.1.1")

		// stm: @VENUSGATEWAY_PROOF_EVENT_LISTEN_PROOF_EVENT_003
		_, err := proof.ListenProofEvent(ctx, &types.ProofRegisterPolicy{
			MinerAddress: addr2,
		})
		require.Contains(t, err.Error(), "verify miner:")
	})

	t.Run("no ip exit", func(t *testing.T) {
		proof := setupProofEvent(t, []address.Address{addr1})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		// stm: @VENUSGATEWAY_PROOF_EVENT_LISTEN_PROOF_EVENT_002
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
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		proof := setupProofEvent(t, []address.Address{addr})
		expectInfo := []builtin.ExtendedSectorInfo{
			{
				SealProof:    abi.RegisteredSealProof_StackedDrg2KiBV1_1,
				SectorNumber: 100,
				SectorKey:    nil,
				SealedCID:    cid.Undef,
			},
		}
		expectRand := []byte{1, 23}
		expectEpoch := abi.ChainEpoch(100)
		expectVersion := network.Version(10)
		expectProof := []builtin.PoStProof{
			{
				PoStProof:  abi.RegisteredPoStProof_StackedDrgWindow32GiBV1,
				ProofBytes: []byte{3, 4},
			},
		}
		handler := testhelper.NewProofHander(t, expectInfo, expectRand, expectEpoch, expectVersion, expectProof, false)
		proofClient := NewProofEvent(proof, addr, handler, log.With())

		go proofClient.ListenProofRequest(core.CtxWithTokenLocation(ctx, "127.1.1.1"))
		proofClient.WaitReady(ctx)

		// stm: @VENUSGATEWAY_PROOF_EVENT_COMPUTE_PROOF_001
		result, err := proof.ComputeProof(ctx, addr, expectInfo, expectRand, expectEpoch, expectVersion)
		require.NoError(t, err)
		require.Equal(t, expectProof, result)
	})

	t.Run("send unmarshal", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		proof := setupProofEvent(t, []address.Address{addr})
		expectInfo := []builtin.ExtendedSectorInfo{
			{
				SealProof:    abi.RegisteredSealProof_StackedDrg2KiBV1_1,
				SectorNumber: 100,
				SectorKey:    nil,
				SealedCID:    cid.Undef,
			},
		}
		expectRand := []byte{1, 23}
		expectEpoch := abi.ChainEpoch(100)
		expectVersion := network.Version(10)
		expectProof := []builtin.PoStProof{
			{
				PoStProof:  abi.RegisteredPoStProof_StackedDrgWindow32GiBV1,
				ProofBytes: []byte{3, 4},
			},
		}
		handler := testhelper.NewProofHander(t, expectInfo, expectRand, expectEpoch, expectVersion, expectProof, false)
		proofClient := NewProofEvent(proof, addr, handler, log.With())

		ctx = core.CtxWithTokenLocation(ctx, "127.1.1.1")
		go proofClient.ListenProofRequest(ctx)
		proofClient.WaitReady(ctx)

		{
			ctx := context.Background()
			channels, err := proof.getChannels(addr)
			require.NoError(t, err)
			var result []builtin.PoStProof
			// stm: @VENUSGATEWAY_PROOF_EVENT_COMPUTE_PROOF_002
			err = proof.SendRequest(ctx, channels, "ComputeProof", []byte{1, 3, 5, 1, 3}, &result)
			require.Contains(t, err.Error(), "invalid character")
		}
	})

	t.Run("response error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		proof := setupProofEvent(t, []address.Address{addr})
		expectInfo := []builtin.ExtendedSectorInfo{
			{
				SealProof:    abi.RegisteredSealProof_StackedDrg2KiBV1_1,
				SectorNumber: 100,
				SectorKey:    nil,
				SealedCID:    cid.Undef,
			},
		}
		expectRand := []byte{1, 23}
		expectEpoch := abi.ChainEpoch(100)
		expectVersion := network.Version(10)
		handler := testhelper.NewProofHander(t, expectInfo, expectRand, expectEpoch, expectVersion, nil, true)
		proofClient := NewProofEvent(proof, addr, handler, log.With())

		go proofClient.ListenProofRequest(core.CtxWithTokenLocation(ctx, "127.1.1.1"))
		proofClient.WaitReady(ctx)

		// stm: @VENUSGATEWAY_PROOF_EVENT_COMPUTE_PROOF_004
		result, err := proof.ComputeProof(ctx, addr, expectInfo, expectRand, expectEpoch, expectVersion)
		require.EqualError(t, err, "mock error")
		require.Nil(t, result)

		// stm: @VENUSGATEWAY_PROOF_EVENT_COMPUTE_PROOF_003
		addr3 := addrGetter()
		_, err = proof.ComputeProof(ctx, addr3, expectInfo, expectRand, expectEpoch, expectVersion)
		require.Error(t, err)
		require.Contains(t, err.Error(), "no connections for this miner")
	})

	t.Run("incorrect result  error", func(t *testing.T) {
		proof := setupProofEvent(t, []address.Address{addr})
		{
			ctx := core.CtxWithTokenLocation(context.Background(), "127.1.1.1")
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
		ctx := core.CtxWithTokenLocation(context.Background(), "127.1.1.1")
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
		proofClient := NewProofEvent(proof, addr1, nil, log.With())
		ctx = core.CtxWithTokenLocation(ctx, "127.1.1.1")
		go proofClient.ListenProofRequest(ctx)
		proofClient.WaitReady(ctx)
	}

	{
		proofClient := NewProofEvent(proof, addr2, nil, log.With())
		ctx = core.CtxWithTokenLocation(ctx, "127.1.1.1")
		go proofClient.ListenProofRequest(ctx)
		proofClient.WaitReady(ctx)
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
		proofClient := NewProofEvent(proof, addr1, nil, log.With())
		ctx = core.CtxWithTokenLocation(ctx, "127.1.1.1")
		go proofClient.ListenProofRequest(ctx)
		proofClient.WaitReady(ctx)
	}

	{
		proofClient := NewProofEvent(proof, addr1, nil, log.With())
		ctx = core.CtxWithTokenLocation(ctx, "127.1.1.2")
		go proofClient.ListenProofRequest(ctx)
		proofClient.WaitReady(ctx)
	}

	// todo: should change 'LISTEN' to 'LIST', it maybe a spell mistake.
	// stm: @VENUSGATEWAY_PROOF_EVENT_LISTEN_CONNECTED_MINERS_001
	connetions, err := proof.ListMinerConnection(ctx, addr1)
	require.NoError(t, err)
	require.Equal(t, connetions.ConnectionCount, 2)
}

func setupProofEvent(t *testing.T, validateAddr []address.Address) *ProofEventStream {
	return NewProofEventStream(context.Background(), &validator.MockAuthMinerValidator{ValidatedAddr: validateAddr}, gtypes.DefaultConfig())
}
