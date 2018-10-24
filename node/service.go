package node

import (
	"github.com/coschain/contentos-go/p2p"
	"reflect"
)

type ServiceContext struct {
	config   *Config
	services map[reflect.Type]Service
}

func (ctx *ServiceContext) ResolvePath(path string) string {
	return ctx.config.ResolvePath(path)
}

func (ctx *ServiceContext) Service(service interface{}) error {
	element := reflect.ValueOf(service).Elem()
	if running, ok := ctx.services[element.Type()]; ok {
		element.Set(reflect.ValueOf(running))
		return nil
	}
	return ErrServiceUnknown
}

type ServiceConstructor func(ctx *ServiceContext) (Service, error)

type Service interface {
	Start(server *p2p.Server) error

	// stop all goroutines belonging to the service,
	// blocking until all of them are terminated.
	Stop() error
}
