package wallet

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/coschain/contentos-go/prototype"
	"github.com/tyler-smith/go-bip32"
	"github.com/tyler-smith/go-bip39"
	"math"
	"math/big"
	"strings"
)

var DefaultRootDerivationPath = DerivationPath{0x80000000 + 44, 0x80000000 + 3077, 0x80000000 + 0, 0, 0}

type DerivationPath []uint32

func ParseDerivationPath(path string) (DerivationPath, error) {
	var result DerivationPath

	// Handle absolute or relative paths
	components := strings.Split(path, "/")
	switch {
	case len(components) == 0:
		return nil, errors.New("empty derivation path")

	case strings.TrimSpace(components[0]) == "":
		return nil, errors.New("ambiguous path: use 'm/' prefix for absolute paths, or no leading '/' for relative ones")

	case strings.TrimSpace(components[0]) == "m":
		components = components[1:]

	default:
		result = append(result, DefaultRootDerivationPath...)
	}
	// All remaining components are relative, append one by one
	if len(components) == 0 {
		return nil, errors.New("empty derivation path") // Empty relative paths
	}
	for _, component := range components {
		// Ignore any user added whitespace
		component = strings.TrimSpace(component)
		var value uint32

		// Handle hardened paths
		if strings.HasSuffix(component, "'") {
			// 2 ^ 31
			value = 0x80000000
			component = strings.TrimSpace(strings.TrimSuffix(component, "'"))
		}
		// Handle the non hardened component
		bigval, ok := new(big.Int).SetString(component, 0)
		if !ok {
			return nil, fmt.Errorf("invalid component: %s", component)
		}
		max := math.MaxUint32 - value
		if bigval.Sign() < 0 || bigval.Cmp(big.NewInt(int64(max))) > 0 {
			if value == 0 {
				return nil, fmt.Errorf("component %v out of allowed range [0, %d]", bigval, max)
			}
			return nil, fmt.Errorf("component %v out of allowed hardened range [0, %d]", bigval, max)
		}
		value += uint32(bigval.Uint64())

		// Append and repeat
		result = append(result, value)
	}
	return result, nil
}

// String implements the stringer interface, converting a binary derivation path
// to its canonical representation.
func (path DerivationPath) String() string {
	result := "m"
	for _, component := range path {
		var hardened bool
		if component >= 0x80000000 {
			component -= 0x80000000
			hardened = true
		}
		result = fmt.Sprintf("%s/%d", result, component)
		if hardened {
			result += "'"
		}
	}
	return result
}

type BaseHDWallet struct {
	BaseWallet
	mnemonic string
	seed []byte
	hdPath string
}

func NewBaseHDWallet(name string, path string) *BaseHDWallet {
	return &BaseHDWallet{
		BaseWallet: BaseWallet{
			name:     name,
			unlocked: make(map[string]*PrivAccount),
			locked:   make(map[string]*EncryptAccount),
			dirPath:  path,
		},
		hdPath: "m/44'/3077'/0'/0/0",
	}
}

func (w *BaseHDWallet) GenerateNewMnemonic() (string, error) {
	entropy, err := bip39.NewEntropy(256)
	if err != nil {
		return "", err
	}
	mnemonic, err := bip39.NewMnemonic(entropy)
	if err != nil {
		return "", err
	}
	return mnemonic, err
}

func (w *BaseHDWallet) GenerateFromMnemonic(mnemonic string) (string, string, error) {
	seed := bip39.NewSeed(mnemonic, "")
	path, err := ParseDerivationPath(w.hdPath)
	if err != nil {
		return "", "", err
	}
	masterKey, err := bip32.NewMasterKey(seed)
	if err != nil {
		return  "", "", err
	}
	key := masterKey
	for _, n := range path {
		key, err = key.NewChildKey(n)
		if err != nil {
			return "", "", err
		}
	}
	sigRawKey := crypto.ToECDSAUnsafe(key.Key)
	privKey := prototype.PrivateKeyFromECDSA(sigRawKey)

	pubKey, err := privKey.PubKey()
	if err != nil {
		return "", "", err
	}
	privKeyStr := privKey.ToWIF()
	pubKeyStr := pubKey.ToWIF()
	return pubKeyStr, privKeyStr, nil
}



