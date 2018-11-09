package wallet

type Wallet interface {
	Name() string

	Path() string

	Create(name, passphrase string) error

	Load(name string) error

	Lock(name string) error

	Unlock(name, passphrase string) error

	List() []string

	Info(name string) string

	Close()

	IsLocked(name string) (bool, error)

	//CheckAccountName(name string) (bool)

}
