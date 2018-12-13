package vm

type RequireAuth func(name string)
type Transfer func(from, to string, amount uint64, memo string)

type Injector interface {
	RequireAuth(name string) error
	Transfer(from, to string, amount uint64, memo string) error
}
