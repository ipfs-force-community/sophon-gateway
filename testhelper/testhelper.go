package testhelper

import (
	"context"
	"fmt"
	"testing"

	"github.com/filecoin-project/specs-storage/storage"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	"github.com/ipfs/go-cid"

	"github.com/ipfs-force-community/venus-gateway/types"

	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
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
	fail bool) types.ProofHandler {
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

func (p *proofhandler) ComputeProof(ctx context.Context, infos []builtin.ExtendedSectorInfo, randomness abi.PoStRandomness, epoch abi.ChainEpoch, version network.Version) ([]builtin.PoStProof, error) {
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
	t               *testing.T
	expectSectorRef storage.SectorRef
	expectOffset    sharedTypes.PaddedByteIndex
	expectSize      abi.PaddedPieceSize

	expectPieceCid cid.Cid
	expectDest     string
	fail           bool
}

func NewMarketHandler(t *testing.T) *MarketHandler {
	return &MarketHandler{t: t}
}

func (p *MarketHandler) SetCheckIsUnsealExpect(s storage.SectorRef, offset sharedTypes.PaddedByteIndex, size abi.PaddedPieceSize, fail bool) {
	p.expectSectorRef = s
	p.expectOffset = offset
	p.expectSize = size
	p.fail = fail
}

func (p *MarketHandler) CheckIsUnsealed(ctx context.Context, s storage.SectorRef, offset sharedTypes.PaddedByteIndex, size abi.PaddedPieceSize) (bool, error) {
	require.Equal(p.t, p.expectSectorRef, s)
	require.Equal(p.t, p.expectOffset, offset)
	require.Equal(p.t, p.expectSize, size)
	if p.fail {
		return false, fmt.Errorf("mock error")
	}
	return true, nil
}

func (p *MarketHandler) SetSectorsUnsealPieceExpect(pieceCid cid.Cid, sector storage.SectorRef, offset sharedTypes.PaddedByteIndex, size abi.PaddedPieceSize, dest string, fail bool) {
	p.expectPieceCid = pieceCid
	p.expectSectorRef = sector
	p.expectOffset = offset
	p.expectSize = size
	p.expectDest = dest
	p.fail = fail
}

func (p *MarketHandler) SectorsUnsealPiece(ctx context.Context, pieceCid cid.Cid, sector storage.SectorRef, offset sharedTypes.PaddedByteIndex, size abi.PaddedPieceSize, dest string) error {
	require.Equal(p.t, p.expectPieceCid, pieceCid)
	require.Equal(p.t, p.expectSectorRef, sector)
	require.Equal(p.t, p.expectOffset, offset)
	require.Equal(p.t, p.expectSize, size)
	require.Equal(p.t, p.expectDest, dest)
	if p.fail {
		return fmt.Errorf("mock error")
	}
	return nil
}
