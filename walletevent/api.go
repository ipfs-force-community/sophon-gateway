package walletevent

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/crypto"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types "github.com/filecoin-project/venus/venus-shared/types/gateway"
)

type IWalletEvent interface {
	ListWalletInfo(ctx context.Context) ([]*types.WalletDetail, error)
	ListWalletInfoByWallet(ctx context.Context, wallet string) (*types.WalletDetail, error)

	WalletHas(ctx context.Context, supportAccount string, addr address.Address) (bool, error)
	WalletSign(ctx context.Context, account string, addr address.Address, toSign []byte, meta sharedTypes.MsgMeta) (*crypto.Signature, error)
}

type IWalletEventAPI interface {
	ResponseWalletEvent(ctx context.Context, resp *types.ResponseEvent) error
	ListenWalletEvent(ctx context.Context, policy *types.WalletRegisterPolicy) (<-chan *types.RequestEvent, error)
	SupportNewAccount(ctx context.Context, channelId sharedTypes.UUID, account string) error
	AddNewAddress(ctx context.Context, channelId sharedTypes.UUID, newAddrs []address.Address) error
	RemoveAddress(ctx context.Context, channelId sharedTypes.UUID, newAddrs []address.Address) error
}
