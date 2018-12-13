package app

type RequireAuth func(name string)
type Transfer func(from, to string, amount uint64, memo string)

type Injector struct {
	RequireAuth
	Transfer
}
