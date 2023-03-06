package testhelper

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"

	"github.com/ipfs-force-community/venus-gateway/types"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/crypto"
	vcrypto "github.com/filecoin-project/venus/pkg/crypto"
	_ "github.com/filecoin-project/venus/pkg/crypto/secp"
	"github.com/filecoin-project/venus/pkg/wallet/key"
	sharedTypes "github.com/filecoin-project/venus/venus-shared/types"
)

var _ types.IWalletHandler = (*MemWallet)(nil)

type MemWallet struct {
	lk   sync.Mutex
	keys map[address.Address]key.KeyInfo
	fail bool
}

func NewMemWallet() *MemWallet {
	return &MemWallet{
		lk:   sync.Mutex{},
		keys: make(map[address.Address]key.KeyInfo),
	}
}

func (m *MemWallet) SetFail(ctx context.Context, fail bool) {
	m.fail = fail
}

func (m *MemWallet) GetKey(ctx context.Context, addr address.Address) (key.KeyInfo, error) {
	m.lk.Lock()
	defer m.lk.Unlock()
	keyInfo, ok := m.keys[addr]
	if ok {
		return keyInfo, nil
	}
	return key.KeyInfo{}, fmt.Errorf("key not found")
}

func (m *MemWallet) AddKey(ctx context.Context) (address.Address, error) {
	m.lk.Lock()
	defer m.lk.Unlock()
	keyInfo, err := key.NewSecpKeyFromSeed(rand.Reader)
	if err != nil {
		return address.Undef, err
	}

	addr, err := keyInfo.Address()
	if err != nil {
		return addr, err
	}
	m.keys[addr] = keyInfo
	return addr, nil
}

func (m *MemWallet) AddDelegatedKey(ctx context.Context) (address.Address, error) {
	m.lk.Lock()
	defer m.lk.Unlock()

	keyInfo, err := key.NewDelegatedKeyFromSeed(rand.Reader)
	if err != nil {
		return address.Undef, err
	}
	addr, err := keyInfo.Address()
	if err != nil {
		return addr, err
	}

	m.keys[addr] = keyInfo
	return addr, nil
}

func (m *MemWallet) Verify(ctx context.Context, addr address.Address, sig *crypto.Signature, msg []byte) error {
	return vcrypto.Verify(sig, addr, msg)
}

func (m *MemWallet) WalletList(ctx context.Context) ([]address.Address, error) {
	if m.fail {
		return nil, fmt.Errorf("mock error")
	}

	m.lk.Lock()
	defer m.lk.Unlock()
	var result []address.Address
	for _, keyInfo := range m.keys {
		addr, err := keyInfo.Address()
		if err != nil {
			return nil, err
		}
		result = append(result, addr)
	}
	return result, nil
}

func (m *MemWallet) WalletSign(ctx context.Context, signer address.Address, toSign []byte, meta sharedTypes.MsgMeta) (*crypto.Signature, error) {
	if m.fail {
		return nil, fmt.Errorf("mock error")
	}

	m.lk.Lock()
	defer m.lk.Unlock()
	keyInfo, ok := m.keys[signer]
	if !ok {
		return nil, fmt.Errorf("address %s not found", signer)
	}
	return vcrypto.Sign(toSign, keyInfo.Key(), vcrypto.SigTypeSecp256k1)
}
