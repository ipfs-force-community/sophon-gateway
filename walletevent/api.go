package walletevent

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/venus-wallet/core"
	"github.com/ipfs-force-community/venus-gateway/types"
)

type IWalletEvent interface {
	ListWalletInfo(ctx context.Context) ([]*WalletDetail, error)
	ListWalletInfoByWallet(ctx context.Context, wallet string) (*WalletDetail, error)

	WalletHas(ctx context.Context, supportAccount string, addr address.Address) (bool, error)
	WalletSign(ctx context.Context, account string, addr address.Address, toSign []byte, meta core.MsgMeta) (*crypto.Signature, error)
}

type IWalletEventAPI interface {
	ResponseWalletEvent(ctx context.Context, resp *types.ResponseEvent) error
	ListenWalletEvent(ctx context.Context, supportAccounts []string) (chan *types.RequestEvent, error)
	SupportNewAccount(ctx context.Context, channelId, account string) error
}

var _ IWalletEventAPI = (*WalletEventAPI)(nil)

type WalletEventAPI struct {
	walletEvent *WalletEventStream
}

func NewWalletEventAPI(walletEvent *WalletEventStream) *WalletEventAPI {
	return &WalletEventAPI{walletEvent: walletEvent}
}

func (w *WalletEventAPI) ResponseWalletEvent(ctx context.Context, resp *types.ResponseEvent) error {
	return w.walletEvent.ResponseEvent(ctx, resp)
}

func (w *WalletEventAPI) ListenWalletEvent(ctx context.Context, supportAccounts []string) (chan *types.RequestEvent, error) {
	return w.walletEvent.ListenWalletEvent(ctx, supportAccounts)
}

func (w *WalletEventAPI) SupportNewAccount(ctx context.Context, channelId, account string) error {
	return w.walletEvent.SupportNewAccount(ctx, channelId, account)
}
