package wallet

type Wallet interface {
	Name() string

	Path() string

	GenerateNewKey() (string, string, error) // return pubKey, privKey, error

	Create(name, passphrase, pubKeyStr, privKeyStr string) error

	GetUnlockedAccount(name string) (*PrivAccount, bool)

	Load(name string) error

	Lock(name string) error

	Unlock(name, passphrase string) error

	List() []string

	Info(name string) string

	Close()

	IsLocked(name string) bool

	IsExist(name string) bool
}

type HDWallet interface {
	Wallet

	GenerateNewMnemonic() (string, error)

	CreateFromMnemonic(name, passphrase, mnemonic string) error
}
