package vm

import (
	"github.com/coschain/contentos-go/node"
	"github.com/inconshreveable/log15"
)

type WasmVmService struct {
	ctx           *node.ServiceContext
	registerFuncs map[string]interface{}
	logger        log15.Logger
}

func New(ctx *node.ServiceContext) (*WasmVmService, error) {
	return &WasmVmService{ctx: ctx, registerFuncs: make(map[string]interface{}), logger: log15.New()}, nil
}

func (w *WasmVmService) Run(ctx *Context) (uint32, error) {
	cosVM := NewCosVM(ctx, w.logger)
	for funcName, function := range w.registerFuncs {
		cosVM.Register(funcName, function)
	}
	return cosVM.Run()
}

func (w *WasmVmService) Register(funcName string, function interface{}) {
	w.registerFuncs[funcName] = function
}

func (w *WasmVmService) Start(node *node.Node) error {
	return nil
}

func (w *WasmVmService) Stop() error {
	return nil
}
