package walletevent

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/go-address"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
	types2 "github.com/filecoin-project/venus/venus-shared/types/gateway"
	"github.com/ipfs-force-community/sophon-gateway/types"
)

type walletChannelInfo struct {
	*types.ChannelInfo
	addrs map[address.Address]struct{} // signer address
	// a slice byte provide by wallet, using to verify address is really exist
	signBytes []byte
}

func newWalletChannelInfo(channelInfo *types.ChannelInfo, addrs []address.Address, signBytes []byte) *walletChannelInfo {
	walletInfo := &walletChannelInfo{ChannelInfo: channelInfo, addrs: make(map[address.Address]struct{}), signBytes: signBytes}
	for _, addr := range addrs {
		walletInfo.addrs[addr] = struct{}{}
	}
	return walletInfo
}

type WalletInfo struct {
	walletAccount   string
	supportAccounts map[string]struct{}
	connections     map[sharedTypes.UUID]*walletChannelInfo
}

type IWalletConnMgr interface {
	addNewConn(string, *types2.WalletRegisterPolicy, *walletChannelInfo) error
	getConn(walletAccount string, channelID sharedTypes.UUID) (*walletChannelInfo, error)
	removeConn(string, *walletChannelInfo) error
	addSupportAccount(string, string) error
	getChannels(string, address.Address) ([]*types.ChannelInfo, error)
	addNewAddress(walletAccount string, channelId sharedTypes.UUID, addrs []address.Address) error
	removeAddress(walletAccount string, channelId sharedTypes.UUID, addrs []address.Address) error
	hasWalletChannel(supportAccount string, from address.Address) (bool, error)

	listWalletInfo(ctx context.Context) ([]*types2.WalletDetail, error)
	listWalletInfoByWallet(ctx context.Context, wallet string) (*types2.WalletDetail, error)
}

var _ IWalletConnMgr = (*walletConnMgr)(nil)

type walletConnMgr struct {
	infoLk      sync.Mutex // todo a big lock here , maybe need a smaller lock
	walletInfos map[string]*WalletInfo
}

func newWalletConnMgr() *walletConnMgr {
	return &walletConnMgr{
		infoLk:      sync.Mutex{},
		walletInfos: make(map[string]*WalletInfo),
	}
}

func (w *walletConnMgr) addNewConn(walletAccount string, policy *types2.WalletRegisterPolicy, channel *walletChannelInfo) error {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	var walletInfo *WalletInfo
	var ok bool
	if walletInfo, ok = w.walletInfos[walletAccount]; ok {
		walletInfo.connections[channel.ChannelId] = channel
		for _, supportAccount := range policy.SupportAccounts {
			_, ok := walletInfo.supportAccounts[supportAccount]
			if !ok {
				walletInfo.supportAccounts[supportAccount] = struct{}{}
			}
		}
	} else {
		walletInfo = &WalletInfo{
			walletAccount:   walletAccount,
			supportAccounts: make(map[string]struct{}),
			connections:     map[sharedTypes.UUID]*walletChannelInfo{channel.ChannelId: channel},
		}

		// The supported accounts should include the account corresponding to the token, and it would be absurd not to support yourself!
		walletInfo.supportAccounts[walletAccount] = struct{}{}
		for _, supportAccount := range policy.SupportAccounts {
			if supportAccount != walletAccount {
				walletInfo.supportAccounts[supportAccount] = struct{}{}
			}
		}
		w.walletInfos[walletAccount] = walletInfo
	}

	log.Infow("add wallet connection", "channel", channel.ChannelId.String(),
		"walletName", walletAccount,
		"addrs", channel.addrs,
		"support", walletInfo.supportAccounts,
		"signBytes", policy.SignBytes,
	)
	return nil
}

func (w *walletConnMgr) getConn(walletAccount string, channelID sharedTypes.UUID) (*walletChannelInfo, error) {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	if walletInfo, ok := w.walletInfos[walletAccount]; ok {
		if conn, ok := walletInfo.connections[channelID]; ok {
			return conn, nil
		}
	}

	return nil, fmt.Errorf("no connect found for wallet %s and channelID %s", walletAccount, channelID)
}

func (w *walletConnMgr) removeConn(walletAccount string, info *walletChannelInfo) error {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	if walletInfo, ok := w.walletInfos[walletAccount]; ok {
		delete(walletInfo.connections, info.ChannelId)
		if len(walletInfo.connections) == 0 {
			delete(w.walletInfos, walletAccount)
		}
	}

	log.Infof("wallet %v remove connection %s", walletAccount, info.ChannelId)
	return nil
}

func (w *walletConnMgr) addSupportAccount(walletAccount string, supportAccount string) error {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	if walletInfo, ok := w.walletInfos[walletAccount]; ok {
		walletInfo.supportAccounts[supportAccount] = struct{}{}
	}
	return nil
}

func (w *walletConnMgr) getChannels(supportAccount string, from address.Address) ([]*types.ChannelInfo, error) {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	var channels []*types.ChannelInfo
	for _, walletInfo := range w.walletInfos {
		if _, ok := walletInfo.supportAccounts[supportAccount]; ok {
			for _, conn := range walletInfo.connections {
				if _, ok = conn.addrs[from]; ok {
					channels = append(channels, conn.ChannelInfo)
				}
			}
		}
	}
	if len(channels) == 0 {
		return nil, fmt.Errorf("no connect found for account %s and from %s", supportAccount, from)
	}
	return channels, nil
}

func (w *walletConnMgr) hasWalletChannel(supportAccount string, from address.Address) (bool, error) {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	for _, walletInfo := range w.walletInfos {
		if _, ok := walletInfo.supportAccounts[supportAccount]; ok {
			for _, conn := range walletInfo.connections {
				if _, ok = conn.addrs[from]; ok {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (w *walletConnMgr) addNewAddress(walletAccount string, channelId sharedTypes.UUID, addrs []address.Address) error {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	if walletInfo, ok := w.walletInfos[walletAccount]; ok {
		if channel, ok := walletInfo.connections[channelId]; ok {
			for _, addr := range addrs {
				channel.addrs[addr] = struct{}{}
			}
		} else {
			return fmt.Errorf("channel %s not found ", channelId.String())
		}
	}
	return nil
}

func (w *walletConnMgr) removeAddress(walletAccount string, channelId sharedTypes.UUID, addrs []address.Address) error {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	if walletInfo, ok := w.walletInfos[walletAccount]; ok {
		if channel, ok := walletInfo.connections[channelId]; ok {
			for _, addr := range addrs {
				delete(channel.addrs, addr)
			}
		} else {
			return fmt.Errorf("channel %s not found ", channelId.String())
		}
	}
	return nil
}

func (w *walletConnMgr) listWalletInfo(ctx context.Context) ([]*types2.WalletDetail, error) {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	var walletDetails []*types2.WalletDetail
	for walletAccount, walletInfo := range w.walletInfos {
		walletDetail := &types2.WalletDetail{}
		walletDetail.Account = walletAccount
		for account := range walletInfo.supportAccounts {
			walletDetail.SupportAccounts = append(walletDetail.SupportAccounts, account)
		}
		walletDetail.ConnectStates = []types2.ConnectState{}
		for channelId, wallet := range walletInfo.connections {
			var addrs []address.Address
			for addr := range wallet.addrs {
				addrs = append(addrs, addr)
			}
			cstate := types2.ConnectState{
				Addrs:        addrs,
				ChannelID:    channelId,
				RequestCount: len(wallet.OutBound),
				IP:           wallet.Ip,
				CreateTime:   wallet.CreateTime,
			}
			walletDetail.ConnectStates = append(walletDetail.ConnectStates, cstate)
		}
		walletDetails = append(walletDetails, walletDetail)
	}
	return walletDetails, nil
}

func (w *walletConnMgr) listWalletInfoByWallet(ctx context.Context, wallet string) (*types2.WalletDetail, error) {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	if walletInfo, ok := w.walletInfos[wallet]; ok {
		walletDetail := &types2.WalletDetail{}
		walletDetail.Account = walletInfo.walletAccount
		for account := range walletInfo.supportAccounts {
			walletDetail.SupportAccounts = append(walletDetail.SupportAccounts, account)
		}
		walletDetail.ConnectStates = []types2.ConnectState{}
		for channelId, wallet := range walletInfo.connections {
			var addrs []address.Address
			for addr := range wallet.addrs {
				addrs = append(addrs, addr)
			}
			cstate := types2.ConnectState{
				Addrs:        addrs,
				ChannelID:    channelId,
				IP:           wallet.Ip,
				RequestCount: len(wallet.OutBound),
				CreateTime:   wallet.CreateTime,
			}
			walletDetail.ConnectStates = append(walletDetail.ConnectStates, cstate)
		}
		return walletDetail, nil
	}
	return nil, fmt.Errorf("wallet %s not exit", wallet)
}
