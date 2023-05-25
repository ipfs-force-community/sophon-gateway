package types

import (
	"context"

	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/go-state-types/network"

	"github.com/filecoin-project/venus/venus-shared/actors/builtin"
	"github.com/filecoin-project/venus/venus-shared/types"
)

type ProofHandler interface {
	ComputeProof(context.Context, []builtin.ExtendedSectorInfo, abi.PoStRandomness, abi.ChainEpoch, network.Version) ([]builtin.PoStProof, error)
}

type MarketHandler interface {
	SectorsUnsealPiece(ctx context.Context, miner address.Address, pieceCid cid.Cid, sid abi.SectorNumber, offset types.UnpaddedByteIndex, size abi.UnpaddedPieceSize, dest string) error
}

type IWalletHandler interface {
	WalletList(ctx context.Context) ([]address.Address, error)
	WalletSign(ctx context.Context, signer address.Address, toSign []byte, meta types.MsgMeta) (*crypto.Signature, error)
}
