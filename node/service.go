package node

import (
	"github.com/coschain/contentos-go/p2p"
)

type ServiceContext struct {
	config *Config
	//services map[reflect.Type]Service
	services map[string]Service
}

func (ctx *ServiceContext) ResolvePath(path string) string {
	return ctx.config.ResolvePath(path)
}

//func (ctx *ServiceContext) Service(name string, service interface{}) error {
func (ctx *ServiceContext) Service(name string) (interface{}, error) {
	//element := reflect.ValueOf(service).Elem()
	for k, _ := range ctx.services {
		ctx.config.Logger.Info("ctx service:" + k)
	}
	if running, ok := ctx.services[name]; ok {
		//element.Set(reflect.ValueOf(running))
		//return nil
		return running, nil
	}
	//return ErrServiceUnknown
	return nil, ErrServiceUnknown
}

//func (ctx *ServiceContext) ServiceFromString(serviceName string) (Service, error) {
//	if running, ok := ctx.services[serviceName]; ok {
//		return running, nil
//	}
//	return nil, ErrServiceUnknown
//}

type ServiceConstructor func(ctx *ServiceContext) (Service, error)

type Service interface {
	Start(server *p2p.Server) error

	// stop all goroutines belonging to the service,
	// blocking until all of them are terminated.
	Stop() error
}
