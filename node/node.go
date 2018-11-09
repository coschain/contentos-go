package node

import (
	"errors"
	"github.com/asaskevich/EventBus"
	"github.com/coschain/contentos-go/common/eventloop"
	log "github.com/inconshreveable/log15"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"fmt"
)

// Node is a container and manager of services
type Node struct {
	config *Config

	MainLoop 	*eventloop.EventLoop
	EvBus 		EventBus.Bus

	services     map[string]Service
	serviceFuncs []NamedServiceConstructor // registered services store into this slice

	httpEndpoint string // HTTP endpoint(host + port) to listen at

	stop chan struct{}
	lock sync.RWMutex

	log log.Logger
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
	if conf.Logger == nil {
		conf.Logger = log.New()
	}
	return &Node{
		config:       conf,
		serviceFuncs: []NamedServiceConstructor{},
		httpEndpoint: conf.HTTPEndpoint(),
		log:          conf.Logger,
	}, nil
}

func (n *Node) Register(name string, constructor ServiceConstructor) error {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.serviceFuncs = append(n.serviceFuncs, NamedServiceConstructor{name: name, constructor: constructor})
	return nil
}

func (n *Node) Start() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	if err := n.openDataDir(); err != nil {
		return err
	}

	// which confs should be assigned to p2p configuration

	n.MainLoop 	= eventloop.NewEventLoop()
	n.EvBus		= EventBus.New()

	services := make(map[string]Service)

	for _, namedConstructor := range n.serviceFuncs {
		ctx := &ServiceContext{
			config: n.config,
			// to support services to share, the list of services pass by reference
			services: services,
		}

		name := namedConstructor.name
		constructor := namedConstructor.constructor

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
	for kind, service := range services {
		if err := service.Start(n); err != nil {
			for _, kind := range started {
				services[kind].Stop()
			}

			return err
		}
		started = append(started, kind)
	}

	// start http server
	if err := n.startHTTP(n.config.HTTPEndpoint()); err != nil {
		for _, kind := range started {
			services[kind].Stop()
		}
		return err
	}

	n.services = services
	n.stop = make(chan struct{})

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

// should keep pace with eth even the way of router?
func (n *Node) startHTTP(endpoint string) error {
	return nil
}

func (n *Node) stopHTTP() {
}

func (n *Node) Stop() error {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.stopHTTP()

	failure := &StopError{
		Services: make(map[string]error),
	}

	for kind, service := range n.services {
		if err := service.Stop(); err != nil {
			failure.Services[kind] = err
		}
	}
	n.services = nil

	close(n.stop)

	if len(failure.Services) > 0 {
		return failure
	}

	return nil
}

func (n *Node) Wait() {
	n.lock.RLock()

	stop := n.stop
	n.lock.RUnlock()
	<-stop
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
