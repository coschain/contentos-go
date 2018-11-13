package wallet

type Wallet interface {
	Name() string

	SetChainAccount(localName, chainAccountName string) error

	ChainAccount(localName string) string

	Path() string

	GenerateNewKey() (string, string, error) // return pubKey, privKey, error

	Create(name, passphrase, pubKeyStr, privKeyStr string) error

	Load(name string) error

	Lock(name string) error

	Unlock(name, passphrase string) error

	List() []string

	Info(name string) string

	Close()

	IsLocked(name string) (bool, error)

	//CheckAccountName(name string) (bool)

}
