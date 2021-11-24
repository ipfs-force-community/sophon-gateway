package api

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	proof5 "github.com/filecoin-project/specs-actors/v5/actors/runtime/proof"
	"github.com/google/uuid"

	"github.com/ipfs-force-community/venus-gateway/proofevent"
	"github.com/ipfs-force-community/venus-gateway/types"
	"github.com/ipfs-force-community/venus-gateway/types/wallet"
	"github.com/ipfs-force-community/venus-gateway/walletevent"
)

type FullStruct struct {
	IProofEventStruct
	IWalletEvent
}

type IProofEventStruct struct {
	ListConnectedMiners func(ctx context.Context) ([]address.Address, error)                                                                                   `perm:"admin"`
	ListMinerConnection func(ctx context.Context, addr address.Address) (*proofevent.MinerState, error)                                                        `perm:"admin"`
	ComputeProof        func(ctx context.Context, miner address.Address, sectorInfos []proof5.SectorInfo, rand abi.PoStRandomness) ([]proof5.PoStProof, error) `perm:"admin"`

	ResponseProofEvent func(ctx context.Context, resp *types.ResponseEvent) error                                          `perm:"read"`
	ListenProofEvent   func(ctx context.Context, policy *proofevent.ProofRegisterPolicy) (chan *types.RequestEvent, error) `perm:"read"`
}

type IWalletEvent struct {
	ListWalletInfo         func(ctx context.Context) ([]*walletevent.WalletDetail, error)                                                                 `perm:"admin"`
	ListWalletInfoByWallet func(ctx context.Context, wallet string) (*walletevent.WalletDetail, error)                                                    `perm:"admin"`
	WalletHas              func(ctx context.Context, supportAccount string, addr address.Address) (bool, error)                                           `perm:"admin"`
	WalletSign             func(ctx context.Context, account string, addr address.Address, toSign []byte, meta wallet.MsgMeta) (*crypto.Signature, error) `perm:"admin"`

	ResponseWalletEvent func(ctx context.Context, resp *types.ResponseEvent) error                                            `perm:"read"`
	ListenWalletEvent   func(ctx context.Context, policy *walletevent.WalletRegisterPolicy) (chan *types.RequestEvent, error) `perm:"read"`
	SupportNewAccount   func(ctx context.Context, channelId uuid.UUID, account string) error                                  `perm:"read"`
	AddNewAddress       func(ctx context.Context, channelId uuid.UUID, newAddrs []address.Address) error                      `perm:"read"`
	RemoveAddress       func(ctx context.Context, channelId uuid.UUID, newAddrs []address.Address) error                      `perm:"read"`
}
