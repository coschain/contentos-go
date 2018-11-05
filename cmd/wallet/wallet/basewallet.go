package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"github.com/coschain/contentos-go/common/type-proto"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type BaseWallet struct {
	name string

	dirPath string

	unlocked map[string]*PrivAccount

	locked map[string]*EncryptAccount

	mu sync.RWMutex
}

//func NewBaseWallet() *BaseWallet {
//	return &BaseWallet{
//		name: "default",
//		unlocked: make(map[string]*PrivAccount),
//		locked: make(map[string]*EncryptAccount),
//	}
//}

func EncryptData(data, key []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte{}, []byte{}, err
	}
	cipherdata := make([]byte, len(data))
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return []byte{}, []byte{}, err
	}
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(cipherdata, data)
	return cipherdata, iv, nil
}

func DecryptData(cipherdata, key, iv []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte{}, err
	}
	if (len(cipherdata) % aes.BlockSize) != 0 {
		return []byte{}, err
	}
	data := make([]byte, len(cipherdata))
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(data, cipherdata)
	return data, nil
}

func (w *BaseWallet) Name() string {
	return w.name
}

func (w *BaseWallet) Path() string {
	return w.dirPath
}

func (w *BaseWallet) LoadDir() error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	var err error
	if _, err = os.Stat(w.dirPath); os.IsNotExist(err) {
		return err
	}
	err = filepath.Walk(w.dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".json") {
			accjson, err := ioutil.ReadFile(path)
			if err != nil {
				return err
			}
			var acc EncryptAccount
			if err := json.Unmarshal(accjson, &acc); err != nil {
				return err
			}
			w.locked[acc.Name] = &acc
		}
		return nil
	})
	return nil
}

func (w *BaseWallet) Create(name, passphrase string) error {
	privKey, err := prototype.GenerateNewKey()
	if err != nil {
		return err
	}
	pubKey, err := privKey.PubKey()
	if err != nil {
		return err
	}
	privKeyStr := privKey.ToWIF()
	pubKeyStr := pubKey.ToWIF()
	cipher_data, iv, err := EncryptData([]byte(privKeyStr), []byte(passphrase))
	encrypt_account := &EncryptAccount{
		Account: Account{Name: name, PubKey: pubKeyStr},
		Cipher:  string(cipher_data),
		Iv:      string(iv),
		Version: 1,
	}
	priv_account := &PrivAccount{
		Account: Account{Name: name, PubKey: pubKeyStr},
		PrivKey: privKeyStr,
	}
	w.locked[name] = encrypt_account
	w.unlocked[name] = priv_account
	w.Seal(encrypt_account)
	return nil
}

// name should not be a path
// todo: check name
func (w *BaseWallet) Load(name string) error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	var filename string
	if strings.HasSuffix(name, ".json") {
		filename = name
	} else {
		filename = name + ".json"
	}
	path := filepath.Join(w.dirPath, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return err
	}
	accjson, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	var acc EncryptAccount
	if err := json.Unmarshal(accjson, &acc); err != nil {
		return err
	}
	w.locked[acc.Name] = &acc
	return nil
}

// w.locked hold all EncryptAccount and never modifies  its content except add new account
// when unlock, a account being decrypted and added into unlock map
func (w *BaseWallet) Lock(name string) error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if _, ok := w.locked[name]; !ok {
		return &UnknownLockedAccountError{Name: name}
	}
	if _, ok := w.unlocked[name]; ok {
		delete(w.unlocked, name)
	}
	return nil
}

func (w *BaseWallet) Unlock(name, passphrase string) error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if _, ok := w.unlocked[name]; ok {
		return &ReentrantUnlockedAccountError{Name: name}
	}
	if encrypt_acc, ok := w.locked[name]; !ok {
		return &UnknownLockedAccountError{Name: name}
	} else {
		key := []byte(passphrase)
		iv := []byte(encrypt_acc.Iv)
		cipher_data := []byte(encrypt_acc.CipherText)
		priv_key, err := DecryptData(cipher_data, key, iv)
		if err != nil {
			return err
		}
		acc := &PrivAccount{Account{Name: name, PubKey: encrypt_acc.PubKey}, string(priv_key)}
		w.unlocked[name] = acc
		return nil
	}
}

func (w *BaseWallet) IsLocked(name string) (bool, error) {
	if _, ok := w.unlocked[name]; ok {
		return false, nil
	}

	if _, ok := w.locked[name]; ok {
		return true, nil
	} else {
		return false, &UnknownLockedAccountError{Name: name}
	}

}

func (w *BaseWallet) Seal(account *EncryptAccount) error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	name := account.Name

	// I knew there is a problem when user create a pair key but using a name which have been occupied.
	// fixme

	filename := fmt.Sprintf("COS-KEYJSON-%s.json", name)
	path := filepath.Join(w.dirPath, filename)
	keyjson, err := json.Marshal(account)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, keyjson, 0600)
	return nil
}
