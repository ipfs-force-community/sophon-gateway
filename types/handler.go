package types

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/network"
	"github.com/filecoin-project/specs-storage/storage"
	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	"github.com/filecoin-project/venus/venus-shared/types"
	"github.com/ipfs/go-cid"
)

type ProofHandler interface {
	ComputeProof(context.Context, []builtin.ExtendedSectorInfo, abi.PoStRandomness, abi.ChainEpoch, network.Version) ([]builtin.PoStProof, error)
}

type MarketHandler interface {
	CheckIsUnsealed(ctx context.Context, s storage.SectorRef, offset types.PaddedByteIndex, size abi.PaddedPieceSize) (bool, error)
	SectorsUnsealPiece(ctx context.Context, pieceCid cid.Cid, sector storage.SectorRef, offset types.PaddedByteIndex, size abi.PaddedPieceSize, dest string) error
}

type IWalletHandler interface {
	WalletList(ctx context.Context) ([]address.Address, error)
	WalletSign(ctx context.Context, signer address.Address, toSign []byte, meta types.MsgMeta) (*crypto.Signature, error)
}
