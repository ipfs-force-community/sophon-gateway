package api

import (
	"context"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"

	"github.com/filecoin-project/specs-storage/storage"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/crypto"
	proof5 "github.com/filecoin-project/specs-actors/v5/actors/runtime/proof"

	types2 "github.com/ipfs-force-community/venus-common-utils/types"
	"github.com/ipfs-force-community/venus-gateway/marketevent"

	"github.com/ipfs-force-community/venus-gateway/proofevent"
	"github.com/ipfs-force-community/venus-gateway/types"

	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	"github.com/ipfs-force-community/venus-gateway/walletevent"
)

type GatewayFullNode interface {
	IProofEvent
	IWalletEvent
	IMarketEvent
}

type IProofEvent interface {
	ListConnectedMiners(ctx context.Context) ([]address.Address, error)                                                                            //perm:admin
	ListMinerConnection(ctx context.Context, addr address.Address) (*proofevent.MinerState, error)                                                 //perm:admin
	ComputeProof(ctx context.Context, miner address.Address, sectorInfos []proof5.SectorInfo, rand abi.PoStRandomness) ([]proof5.PoStProof, error) //perm:admin

	ResponseProofEvent(ctx context.Context, resp *types.ResponseEvent) error                                          //perm:read
	ListenProofEvent(ctx context.Context, policy *proofevent.ProofRegisterPolicy) (<-chan *types.RequestEvent, error) //perm:read
}

type IWalletEvent interface {
	ListWalletInfo(ctx context.Context) ([]*walletevent.WalletDetail, error)                                                                  //perm:admin
	ListWalletInfoByWallet(ctx context.Context, wallet string) (*walletevent.WalletDetail, error)                                             //perm:admin
	WalletHas(ctx context.Context, supportAccount string, addr address.Address) (bool, error)                                                 //perm:admin
	WalletSign(ctx context.Context, account string, addr address.Address, toSign []byte, meta sharedTypes.MsgMeta) (*crypto.Signature, error) //perm:admin

	ResponseWalletEvent(ctx context.Context, resp *types.ResponseEvent) error                                            //perm:read
	ListenWalletEvent(ctx context.Context, policy *walletevent.WalletRegisterPolicy) (<-chan *types.RequestEvent, error) //perm:read
	SupportNewAccount(ctx context.Context, channelId uuid.UUID, account string) error                                    //perm:read
	AddNewAddress(ctx context.Context, channelId uuid.UUID, newAddrs []address.Address) error                            //perm:read
	RemoveAddress(ctx context.Context, channelId uuid.UUID, newAddrs []address.Address) error                            //perm:read
}

type IMarketEvent interface {
	ListMarketConnectionsState(ctx context.Context) ([]marketevent.MarketConnectionState, error)                                                                                           //perm:admin
	IsUnsealed(ctx context.Context, miner address.Address, pieceCid cid.Cid, sector storage.SectorRef, offset types2.PaddedByteIndex, size abi.PaddedPieceSize) (bool, error)              //perm:admin
	SectorsUnsealPiece(ctx context.Context, miner address.Address, pieceCid cid.Cid, sector storage.SectorRef, offset types2.PaddedByteIndex, size abi.PaddedPieceSize, dest string) error //perm:admin

	ResponseMarketEvent(ctx context.Context, resp *types.ResponseEvent) error                                            //perm:read
	ListenMarketEvent(ctx context.Context, policy *marketevent.MarketRegisterPolicy) (<-chan *types.RequestEvent, error) //perm:read
}
