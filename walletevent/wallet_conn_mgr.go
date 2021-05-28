package walletevent

import (
	"context"
	"github.com/filecoin-project/go-address"
	"github.com/google/uuid"
	"github.com/ipfs-force-community/venus-gateway/types"
	"golang.org/x/xerrors"
	"sync"
)

type walletChannelInfo struct {
	*types.ChannelInfo
	addrs map[address.Address]struct{}
}

func newWalletChannelInfo(channelInfo *types.ChannelInfo, addrs []address.Address) *walletChannelInfo {
	walletInfo := &walletChannelInfo{ChannelInfo: channelInfo, addrs: make(map[address.Address]struct{})}
	for _, addr := range addrs {
		walletInfo.addrs[addr] = struct{}{}
	}
	return walletInfo
}

type WalletInfo struct {
	WalletAccount   string
	SupportAccounts map[string]struct{}
	Connections     map[uuid.UUID]*walletChannelInfo
}

type IWalletConnMgr interface {
	AddNewConn(string, []string, []address.Address, *walletChannelInfo) error
	RemoveConn(string, *walletChannelInfo) error
	AddSupportAccount(string, string) error
	GetChannels(string, address.Address) ([]*types.ChannelInfo, error)
	NewAddress(walletAccount string, channelId uuid.UUID, addrs []address.Address) error
	HasWalletChannel(supportAccount string, from address.Address) (bool, error)

	ListWalletInfo(ctx context.Context) ([]*WalletDetail, error)
	ListWalletInfoByWallet(ctx context.Context, wallet string) (*WalletDetail, error)
}

var _ IWalletConnMgr = (*walletConnMgr)(nil)

type walletConnMgr struct {
	infoLk      sync.Mutex //todo a big lock here , maybe need a smaller lock
	walletInfos map[string]*WalletInfo
}

func newWalletConnMgr() *walletConnMgr {
	return &walletConnMgr{
		infoLk:      sync.Mutex{},
		walletInfos: make(map[string]*WalletInfo),
	}
}

func (w *walletConnMgr) AddNewConn(walletAccount string, accounts []string, addrs []address.Address, channel *walletChannelInfo) error {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	var walletInfo *WalletInfo
	var ok bool
	if walletInfo, ok = w.walletInfos[walletAccount]; ok {
		walletInfo.Connections[channel.ChannelId] = channel
		for _, supportAccount := range accounts {
			_, ok := walletInfo.SupportAccounts[supportAccount]
			if !ok {
				walletInfo.SupportAccounts[supportAccount] = struct{}{}
			}
		}
	} else {
		walletInfo = &WalletInfo{
			WalletAccount:   walletAccount,
			SupportAccounts: make(map[string]struct{}),
			Connections:     map[uuid.UUID]*walletChannelInfo{channel.ChannelId: channel},
		}

		for _, supportAccount := range accounts {
			walletInfo.SupportAccounts[supportAccount] = struct{}{}
		}
		w.walletInfos[walletAccount] = walletInfo
	}

	log.Infow("add wallet connection", "channel", channel.ChannelId.String(),
		"walletName", walletAccount,
		"addrs", addrs,
		"support", walletInfo.SupportAccounts,
	)
	return nil

}

func (w *walletConnMgr) RemoveConn(walletAccount string, info *walletChannelInfo) error {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	if walletInfo, ok := w.walletInfos[walletAccount]; ok {
		delete(walletInfo.Connections, info.ChannelId)
		if len(walletInfo.Connections) == 0 {
			delete(w.walletInfos, walletAccount)
		}
	}

	log.Infof("wallet %v remove connection %s", walletAccount, info.ChannelId)
	return nil
}

func (w *walletConnMgr) AddSupportAccount(walletAccount string, supportAccount string) error {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	if walletInfo, ok := w.walletInfos[walletAccount]; ok {
		walletInfo.SupportAccounts[supportAccount] = struct{}{}
	}
	return nil
}

func (w *walletConnMgr) GetChannels(supportAccount string, from address.Address) ([]*types.ChannelInfo, error) {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	var channels []*types.ChannelInfo
	for _, walletInfo := range w.walletInfos {
		if _, ok := walletInfo.SupportAccounts[supportAccount]; ok {
			for _, conn := range walletInfo.Connections {
				if _, ok = conn.addrs[from]; ok {
					channels = append(channels, conn.ChannelInfo)
				}
			}
		}
	}
	if len(channels) == 0 {
		return nil, xerrors.Errorf("no connect found for account %s and from %s", supportAccount, from)
	}
	return channels, nil
}

func (w *walletConnMgr) HasWalletChannel(supportAccount string, from address.Address) (bool, error) {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	for _, walletInfo := range w.walletInfos {
		if _, ok := walletInfo.SupportAccounts[supportAccount]; ok {
			for _, conn := range walletInfo.Connections {
				if _, ok = conn.addrs[from]; ok {
					return true, nil
				}
			}
		}
	}
	return false, nil
}

func (w *walletConnMgr) NewAddress(walletAccount string, channelId uuid.UUID, addrs []address.Address) error {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	if walletInfo, ok := w.walletInfos[walletAccount]; ok {
		if channel, ok := walletInfo.Connections[channelId]; ok {
			for _, addr := range addrs {
				channel.addrs[addr] = struct{}{}
			}
		} else {
			return xerrors.Errorf("channel %s not found ", channelId.String())
		}
	}
	return nil
}

func (w *walletConnMgr) ListWalletInfo(ctx context.Context) ([]*WalletDetail, error) {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	var walletDetails []*WalletDetail
	for walletAccount, walletInfo := range w.walletInfos {
		walletDetail := &WalletDetail{}
		walletDetail.Account = walletAccount
		for account, _ := range walletInfo.SupportAccounts {
			walletDetail.SupportAccounts = append(walletDetail.SupportAccounts, account)
		}
		walletDetail.ConnectStates = []ConnectState{}
		for channelId, wallet := range walletInfo.Connections {
			var addrs []address.Address
			for addr, _ := range wallet.addrs {
				addrs = append(addrs, addr)
			}
			cstate := ConnectState{
				Addrs:        addrs,
				ChannelId:    channelId,
				RequestCount: len(wallet.OutBound),
				Ip:           wallet.Ip,
				CreateTime:   wallet.CreateTime,
			}
			walletDetail.ConnectStates = append(walletDetail.ConnectStates, cstate)
		}
		walletDetails = append(walletDetails, walletDetail)
	}
	return walletDetails, nil
}

func (w *walletConnMgr) ListWalletInfoByWallet(ctx context.Context, wallet string) (*WalletDetail, error) {
	w.infoLk.Lock()
	defer w.infoLk.Unlock()

	if walletInfo, ok := w.walletInfos[wallet]; ok {
		walletDetail := &WalletDetail{}
		walletDetail.Account = walletInfo.WalletAccount
		for account, _ := range walletInfo.SupportAccounts {
			walletDetail.SupportAccounts = append(walletDetail.SupportAccounts, account)
		}
		walletDetail.ConnectStates = []ConnectState{}
		for channelId, wallet := range walletInfo.Connections {
			var addrs []address.Address
			for addr, _ := range wallet.addrs {
				addrs = append(addrs, addr)
			}
			cstate := ConnectState{
				Addrs:        addrs,
				ChannelId:    channelId,
				Ip:           wallet.Ip,
				RequestCount: len(wallet.OutBound),
				CreateTime:   wallet.CreateTime,
			}
			walletDetail.ConnectStates = append(walletDetail.ConnectStates, cstate)
		}
		return walletDetail, nil
	}
	return nil, xerrors.Errorf("wallet %s not exit", wallet)
}
