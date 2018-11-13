package wallet

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/coschain/contentos-go/prototype"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	PasswordLength    int   = 32
	ExpirationSeconds int64 = 5 * 60
)

type BaseWallet struct {
	name string

	dirPath string

	unlocked map[string]*PrivAccount

	locked map[string]*EncryptAccount

	ticker *time.Ticker

	mu sync.RWMutex
}

func NewBaseWallet(name string, path string) *BaseWallet {
	return &BaseWallet{
		name:     name,
		unlocked: make(map[string]*PrivAccount),
		locked:   make(map[string]*EncryptAccount),
		dirPath:  path,
	}
}

func selectAESAlgorithm(length int) string {
	switch length {
	case 16:
		return "AES-128"
	case 24:
		return "AES-192"
	case 32:
		return "AES-256"
	default:
		break
	}
	return "UNKNOWN"
}

func hashPassphraseToFixLength(input []byte) []byte {
	sha_256 := sha256.New()
	sha_256.Write(input)
	result := sha_256.Sum(nil)
	return result[:PasswordLength]
}

func generateFilename(name string) string {
	filename := fmt.Sprintf("COS-KEYJSON-%s.json", name)
	return filename
}

func EncryptData(data, passphrase []byte) ([]byte, []byte, error) {
	key := hashPassphraseToFixLength(passphrase)
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

func DecryptData(cipherdata, passphrase, iv []byte) ([]byte, error) {
	key := hashPassphraseToFixLength(passphrase)
	block, err := aes.NewCipher(key)
	if err != nil {
		return []byte{}, err
	}
	data := make([]byte, len(cipherdata))
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(data, cipherdata)
	return data, nil
}

func (w *BaseWallet) Start() error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	w.ticker = time.NewTicker(1 * time.Minute)
	go func() {
		for range w.ticker.C {
			current := time.Now().Unix()
			for k, v := range w.unlocked {
				if v.Expire < current {
					delete(w.unlocked, k)
					fmt.Println(fmt.Sprintf("%s expired", k))
				}
			}
		}
	}()
	return nil
}

func (w *BaseWallet) Name() string {
	return w.name
}

func (w *BaseWallet) ChainAccount(localName string) string {
	if acc, ok := w.locked[localName]; ok {
		return acc.ChainAccount
	} else {
		return "UNKNOWN"
	}
}

func (w *BaseWallet) SetChainAccount(localName, accountName string) error {
	if acc, ok := w.locked[localName]; ok {
		acc.ChainAccount = accountName
		return nil
	} else {
		return AccountNotFound{LocalName: localName}
	}
}

func (w *BaseWallet) Path() string {
	return w.dirPath
}

func (w *BaseWallet) LoadAll() error {
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

func (w *BaseWallet) GenerateNewKey() (string, string, error) {
	privKey, err := prototype.GenerateNewKey()
	if err != nil {
		return "", "", err
	}
	pubKey, err := privKey.PubKey()
	if err != nil {
		return "", "", err
	}
	privKeyStr := privKey.ToWIF()
	pubKeyStr := pubKey.ToWIF()
	return pubKeyStr, privKeyStr, nil
}

func (w *BaseWallet) Create(name, passphrase, pubKeyStr, privKeyStr string) error {
	//privKey, err := prototype.PrivateKeyFromWIF(privKeyStr)
	//if err != nil {
	//	return err
	//}
	//pubKey, err :=  privKey.PubKey()
	//if err != nil {
	//	return err
	//}
	//pubKeyStr := pubKey.ToWIF()
	cipher_data, iv, err := EncryptData([]byte(privKeyStr), []byte(passphrase))
	if err != nil {
		return err
	}
	cipher_text := base64.StdEncoding.EncodeToString(cipher_data)
	iv_text := base64.StdEncoding.EncodeToString(iv)
	encrypt_account := &EncryptAccount{
		Account:    Account{Name: name, PubKey: pubKeyStr, ChainAccount: ""},
		Cipher:     selectAESAlgorithm(PasswordLength),
		CipherText: cipher_text,
		Iv:         iv_text,
		Version:    1,
	}
	priv_account := &PrivAccount{
		Account: Account{Name: name, PubKey: pubKeyStr, ChainAccount: encrypt_account.ChainAccount},
		PrivKey: privKeyStr,
	}
	w.locked[name] = encrypt_account
	w.unlocked[name] = priv_account
	w.seal(encrypt_account)
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
		filename = generateFilename(name)
	}
	path := filepath.Join(w.dirPath, filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return err
	}
	accjson, err := ioutil.ReadFile(path)
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
		iv, err := base64.StdEncoding.DecodeString(encrypt_acc.Iv)
		if err != nil {
			return err
		}
		cipher_data, err := base64.StdEncoding.DecodeString(encrypt_acc.CipherText)
		if err != nil {
			return err
		}
		priv_key, err := DecryptData(cipher_data, key, iv)
		if err != nil {
			return err
		}
		expiredTime := time.Now().Unix() + ExpirationSeconds
		acc := &PrivAccount{Account{Name: name, PubKey: encrypt_acc.PubKey, ChainAccount: encrypt_acc.ChainAccount},
			string(priv_key), expiredTime}
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

func (w *BaseWallet) updateAccountExpiredTime(name string) {
	w.mu.RLock()
	defer w.mu.RUnlock()
	if acc, ok := w.unlocked[name]; ok {
		acc.Expire = time.Now().Unix() + ExpirationSeconds
	}
}

func (w *BaseWallet) seal(account *EncryptAccount) error {
	w.mu.RLock()
	defer w.mu.RUnlock()
	name := account.Name

	// I knew there is a problem when user create a pair key but using a name which have been occupied.
	// fixme

	filename := generateFilename(name)
	path := filepath.Join(w.dirPath, filename)
	keyjson, err := json.Marshal(account)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(path, keyjson, 0600)
	return nil
}

func (w *BaseWallet) List() []string {
	var lines []string
	for k := range w.locked {
		if _, ok := w.unlocked[k]; ok {
			lines = append(lines, fmt.Sprintf("account:%12s | status: unlocked", k))
		} else {
			lines = append(lines, fmt.Sprintf("account:%12s | status:   locked", k))
		}
	}
	return lines
}

func (w *BaseWallet) Info(name string) string {
	if acc, ok := w.locked[name]; !ok {
		return fmt.Sprintf("unknown account: %s", name)
	} else {
		content := fmt.Sprintf("account: %s\npub_key: %s\n", acc.Name, acc.PubKey)
		if _, ok = w.unlocked[name]; ok {
			content += fmt.Sprintf("status: unlocked")
		} else {
			content += fmt.Sprintf("status: locked")
		}
		return content
	}
}

func (w *BaseWallet) Close() {
	w.ticker.Stop()
	os.Exit(0)
}
