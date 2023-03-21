package testhelper

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	mktypes "github.com/filecoin-project/venus/venus-shared/types/market"

	"github.com/ipfs-force-community/venus-gateway/types"

	"github.com/stretchr/testify/require"
)

var _ types.ProofHandler = (*proofhandler)(nil)

type proofhandler struct {
	t                *testing.T
	expectInfos      []builtin.ExtendedSectorInfo
	expectRandomness abi.PoStRandomness
	expectEpoch      abi.ChainEpoch
	expectVersion    network.Version
	expectProof      []builtin.PoStProof
	fail             bool
}

func NewProofHander(t *testing.T,
	expectInfos []builtin.ExtendedSectorInfo,
	expectRandomness abi.PoStRandomness,
	expectEpoch abi.ChainEpoch,
	expectVersion network.Version,
	expectProof []builtin.PoStProof,
	fail bool,
) types.ProofHandler {
	return &proofhandler{
		t:                t,
		expectInfos:      expectInfos,
		expectRandomness: expectRandomness,
		expectEpoch:      expectEpoch,
		expectVersion:    expectVersion,
		expectProof:      expectProof,
		fail:             fail,
	}
}

func (p *proofhandler) ComputeProof(_ context.Context, infos []builtin.ExtendedSectorInfo, randomness abi.PoStRandomness, epoch abi.ChainEpoch, version network.Version) ([]builtin.PoStProof, error) {
	require.Equal(p.t, p.expectInfos, infos)
	require.Equal(p.t, p.expectRandomness, randomness)
	require.Equal(p.t, p.expectEpoch, epoch)
	require.Equal(p.t, p.expectVersion, version)
	if p.fail {
		return nil, fmt.Errorf("mock error")
	}
	return p.expectProof, nil
}

func (p *proofhandler) ValidateProof(proof []builtin.PoStProof) {
	require.Equal(p.t, p.expectProof, proof)
}

var _ types.MarketHandler = (*MarketHandler)(nil)

type MarketHandler struct {
	t                  *testing.T
	expectMiner        address.Address
	expectSectorNumber abi.SectorNumber
	expectOffset       sharedTypes.PaddedByteIndex
	expectSize         abi.PaddedPieceSize

	expectPieceCid cid.Cid
	expectTransfer mktypes.Transfer
	fail           bool
}

func NewMarketHandler(t *testing.T) *MarketHandler {
	return &MarketHandler{t: t}
}

func (p *MarketHandler) SetCheckIsUnsealExpect(miner address.Address, sid abi.SectorNumber, offset sharedTypes.PaddedByteIndex, size abi.PaddedPieceSize, fail bool) {
	p.expectMiner = miner
	p.expectSectorNumber = sid
	p.expectOffset = offset
	p.expectSize = size
	p.fail = fail
}

func (p *MarketHandler) CheckIsUnsealed(_ context.Context, miner address.Address, sid abi.SectorNumber, offset sharedTypes.PaddedByteIndex, size abi.PaddedPieceSize) (bool, error) {
	require.Equal(p.t, p.expectMiner, miner)
	require.Equal(p.t, p.expectSectorNumber, sid)
	require.Equal(p.t, p.expectOffset, offset)
	require.Equal(p.t, p.expectSize, size)
	if p.fail {
		return false, fmt.Errorf("mock error")
	}
	return true, nil
}

func (p *MarketHandler) SetSectorsUnsealPieceExpect(pieceCid cid.Cid, miner address.Address, sid abi.SectorNumber, offset sharedTypes.PaddedByteIndex, size abi.PaddedPieceSize, transfer mktypes.Transfer, fail bool) {
	p.expectPieceCid = pieceCid
	p.expectMiner = miner
	p.expectSectorNumber = sid
	p.expectOffset = offset
	p.expectSize = size
	p.expectTransfer = transfer
	p.fail = fail
}

func (p *MarketHandler) SectorsUnsealPiece(_ context.Context, miner address.Address, pieceCid cid.Cid, sid abi.SectorNumber, offset sharedTypes.PaddedByteIndex, size abi.PaddedPieceSize, transfer *mktypes.Transfer) error {
	require.Equal(p.t, p.expectPieceCid, pieceCid)
	require.Equal(p.t, p.expectMiner, miner)
	require.Equal(p.t, p.expectSectorNumber, sid)
	require.Equal(p.t, p.expectOffset, offset)
	require.Equal(p.t, p.expectSize, size)
	require.Equal(p.t, p.expectTransfer.Type, transfer.Type)
	require.Equal(p.t, p.expectTransfer.Params, transfer.Params)
	if p.fail {
		return fmt.Errorf("mock error")
	}
	return nil
}

var _ types.ProofHandler = (*timeoutProofHandler)(nil)

type timeoutProofHandler struct {
	waitTime time.Duration
}

func NewTimeoutProofHandler(waitTime time.Duration) types.ProofHandler {
	return &timeoutProofHandler{waitTime: waitTime}
}

func (h *timeoutProofHandler) ComputeProof(context.Context, []builtin.ExtendedSectorInfo, abi.PoStRandomness, abi.ChainEpoch, network.Version) ([]builtin.PoStProof, error) {
	time.Sleep(h.waitTime)
	return nil, nil
}
