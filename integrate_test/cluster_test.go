package integrate

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	"github.com/filecoin-project/venus/venus-shared/api"
	v2API "github.com/filecoin-project/venus/venus-shared/api/gateway/v2"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	"github.com/ipfs-force-community/metrics"
	"github.com/ipfs-force-community/sophon-gateway/config"
	"github.com/ipfs-force-community/sophon-gateway/marketevent"
	"github.com/ipfs-force-community/sophon-gateway/proofevent"
	"github.com/ipfs-force-community/sophon-gateway/testhelper"
	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/stretchr/testify/require"
)

func TestComputeProof(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mAddr, err := address.NewIDAddress(10)
	require.NoError(t, err)

	// build a cluster
	clusterSize := 5
	clients := buildCluster(ctx, t, []address.Address{mAddr}, clusterSize)
	require.Len(t, clients, clusterSize)

	// make handler
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
	proofEvent := proofevent.NewProofEvent(clients[0], mAddr, handler, logging.Logger("test").With())
	go proofEvent.ListenProofRequest(ctx)
	proofEvent.WaitReady(ctx)

	// simulate vsm support multi gateway
	proofEvent2 := proofevent.NewProofEvent(clients[0], mAddr, handler, logging.Logger("test").With())
	go proofEvent2.ListenProofRequest(ctx)
	proofEvent2.WaitReady(ctx)

	// began test
	proof, err := clients[clusterSize-1].ComputeProof(ctx, mAddr, expectInfo, expectRand, expectEpoch, expectVersion)
	require.NoError(t, err)
	require.Equal(t, expectProof, proof)
}

func TestUnseal(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mAddr, err := address.NewIDAddress(10)
	require.NoError(t, err)

	// build a cluster
	clusterSize := 5
	clients := buildCluster(ctx, t, []address.Address{mAddr}, clusterSize)
	require.Len(t, clients, clusterSize)

	// make handler
	handler := testhelper.NewMarketHandler(t)
	marketEvent := marketevent.NewMarketEventClient(clients[0], mAddr, handler, logging.Logger("test").With())
	go marketEvent.ListenMarketRequest(ctx)
	marketEvent.WaitReady(ctx)

	sid := abi.SectorNumber(10)
	size := abi.UnpaddedPieceSize(100)
	offset := sharedTypes.UnpaddedByteIndex(100)
	dest := ""
	pieceCid, err := cid.Decode("bafy2bzaced2kktxdkqw5pey5of3wtahz5imm7ta4ymegah466dsc5fonj73u2")
	require.NoError(t, err)
	handler.SetSectorsUnsealPieceExpect(pieceCid, mAddr, sid, offset, size, dest, false)

	// began test
	_, err = clients[clusterSize-1].SectorsUnsealPiece(ctx, mAddr, pieceCid, sid, offset, size, dest)
	require.NoError(t, err)
}

func buildCluster(ctx context.Context, t *testing.T, verifiedMiner []address.Address, clusterSize int) (ret []v2API.IGateway) {
	ret = make([]v2API.IGateway, 0)

	commonToken := "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJuYW1lIjoiYWRtaW4iLCJwZXJtIjoiYWRtaW4iLCJleHQiOiIifQ.-jOkOMFVDpbWJVkP9qB0Amfwa7jx7gjNKN0X2aUjBR0"

	// build gateway
	for i := 0; i < clusterSize; i++ {
		ret = append(ret, setupGatewayDaemonAndClient(ctx, t, verifiedMiner, commonToken))
		if i != 0 {
			infos, err := ret[0].MemberInfos(ctx)
			require.NoError(t, err)
			// require.Greater(t, len(infos) , 1)
			require.Len(t, infos, i)
			ret[i].Join(ctx, infos[0].Address)
		}
	}

	return
}

func setupGatewayDaemonAndClient(ctx context.Context, t *testing.T, validateMiner []address.Address, commonToken string) v2API.IGateway {
	cfg := &config.Config{
		API:       &config.APIConfig{ListenAddress: fmt.Sprintf("/ip4/127.0.0.1/tcp/%d", availablePort(t))},
		Auth:      &config.AuthConfig{URL: "127.0.0.1:1", Token: commonToken},
		Metrics:   config.DefaultConfig().Metrics,
		Trace:     &metrics.TraceConfig{JaegerTracingEnabled: false},
		RateLimit: &config.RateLimitCofnig{Redis: ""},
		Cluster: &config.ClusterConfig{
			ListenAddress: "127.0.0.1:0",
		},
	}

	tCfg := defaultTestConfig()

	addr, token, err := MockMain(ctx, validateMiner, t.TempDir(), cfg, tCfg)
	require.NoError(t, err)
	host, port, err := net.SplitHostPort(addr.String())
	require.NoError(t, err)

	multiAddr := fmt.Sprintf("/ip4/%s/tcp/%s/", host, port)

	headers := http.Header{}
	headers.Add(api.AuthorizationHeader, "Bearer "+string(token))

	client, closer, err := v2API.DialIGatewayRPC(ctx, multiAddr, string(token), nil)
	require.NoError(t, err)

	go func() {
		select {
		case <-ctx.Done():
			closer()
		}
	}()

	return client
}

func availablePort(t *testing.T) int {
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err)
	defer listener.Close()

	// 获取监听的地址
	return listener.Addr().(*net.TCPAddr).Port
}
