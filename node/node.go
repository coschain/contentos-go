package node

import (
	"errors"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/common/eventloop"
	"github.com/sirupsen/logrus"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fmt"
)

// Node is a container and manager of services
type Node struct {
	config *Config

	MainLoop *eventloop.EventLoop
	EvBus    EventBus.Bus

	serviceNames []string
	services     map[string]Service
	serviceFuncs []NamedServiceConstructor // registered services store into this slice

	//stop chan struct{}
	lock sync.RWMutex

	//log log.Logger
	Log *logrus.Logger

	StartArgs map[string]interface{}
}

type NamedServiceConstructor struct {
	name        string
	constructor ServiceConstructor
}

func New(conf *Config) (*Node, error) {
	// Copy config
	confCopy := *conf
	conf = &confCopy
	if conf.DataDir != "" {
		dir, err := filepath.Abs(conf.DataDir)
		if err != nil {
			return nil, err
		}
		conf.DataDir = dir
	}
	// Ensure that the instance name doesn't cause weird conflicts with
	// other files in the data directory.
	if strings.ContainsAny(conf.Name, `/\`) {
		return nil, errors.New(`Config.Name must not contain '/' or '\'`)
	}
	//if conf.Logger == nil {
	//	conf.Logger = log.New()
	//}

	return &Node{
		config:       conf,
		serviceNames: []string{},
		serviceFuncs: []NamedServiceConstructor{},
		StartArgs: make(map[string]interface{}),
	}, nil
}

func (n *Node) Register(name string, constructor ServiceConstructor) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.serviceFuncs = append(n.serviceFuncs, NamedServiceConstructor{name: name, constructor: constructor})
	return nil
}

func (n *Node) Start() error {
	noArgs := make(map[string]interface{})
	return n.StartWithArgs(noArgs)
}

func (n *Node) StartWithArgs(args map[string]interface{}) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	// which confs should be assigned to p2p configuration

	n.StartArgs = args
	n.MainLoop = eventloop.NewEventLoop()
	n.EvBus = EventBus.New()
	n.services, n.serviceNames = nil, nil

	if err := n.openDataDir(); err != nil {
		return err
	}

	serviceNames := make([]string, 0, len(n.serviceFuncs))
	services := make(map[string]Service)

	for _, namedConstructor := range n.serviceFuncs {
		ctx := &ServiceContext{
			config: n.config,
			// to support services to share, the list of services pass by reference
			services: services,
		}
		ctx.UpdateChainId()

		name := namedConstructor.name
		constructor := namedConstructor.constructor

		serviceNames = append(serviceNames, name)

		service, err := constructor(ctx)
		if err != nil {
			return err
		}
		if _, exists := services[name]; exists {
			return &DuplicateServiceError{Kind: name}
		}
		services[name] = service
	}

	var started []string
	for _, kind := range serviceNames {
		service := services[kind]
		if err := service.Start(n); err != nil {
			for _, kind := range started {
				_ = services[kind].Stop()
			}

			return err
		}
		started = append(started, kind)
	}

	n.services, n.serviceNames = services, serviceNames
	return nil

}

func (n *Node) openDataDir() error {
	if n.config.DataDir == "" {
		return nil
	}

	confdir := filepath.Join(n.config.DataDir, n.config.name())
	if _, err := os.Stat(confdir); os.IsNotExist(err) {
		fmt.Printf("fatal: not be initialized (do `init` first)\n")
		return err
	}

	return nil
}

func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	failure := &StopError{
		Services: make(map[string]error),
	}

	length := len(n.serviceNames)
	for i := range n.serviceNames {
		kind := n.serviceNames[length-1-i]
		service := n.services[kind]
		if err := service.Stop(); err != nil {
			failure.Services[kind] = err
		}
	}
	n.services, n.serviceNames = nil, nil

	if len(failure.Services) > 0 {
		return failure
	}

	return nil
}

func (n *Node) Wait() {
	n.MainLoop.Run()
}

func (n *Node) Restart() error {
	if err := n.Stop(); err != nil {
		return err
	}

	if err := n.Start(); err != nil {
		return err
	}

	return nil
}

func (n *Node) Service(serviceName string) (interface{}, error) {
	n.lock.RLock()
	defer n.lock.RUnlock()

	if running, ok := n.services[serviceName]; ok {
		return running, nil
	}
	return nil, ErrServiceUnknown
}
