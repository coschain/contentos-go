package vm

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm/context"
	"github.com/inconshreveable/log15"
)

var (
	// fixme: the single id should be share with service
	SINGLE_ID int32 = 1
)

type WasmVmService struct {
	ctx           *node.ServiceContext
	registerFuncs map[string]interface{}
	logger        log15.Logger
	db            iservices.IDatabaseService
	globalProps   *prototype.DynamicProperties
}

func (w *WasmVmService) getDb() (iservices.IDatabaseService, error) {
	s, err := w.ctx.Service(iservices.DbServerName)
	if err != nil {
		return nil, err
	}
	db := s.(iservices.IDatabaseService)
	return db, nil
}

func New(ctx *node.ServiceContext) (*WasmVmService, error) {
	return &WasmVmService{ctx: ctx, registerFuncs: make(map[string]interface{}), logger: log15.New()}, nil
}

func (w *WasmVmService) Run(ctx *vmcontext.Context) (uint32, error) {
	cosVM := NewCosVM(ctx, w.db, w.globalProps, w.logger)
	for funcName, function := range w.registerFuncs {
		cosVM.Register(funcName, function)
	}
	ret, err := cosVM.Run()
	if err != nil {
		w.logger.Error(fmt.Sprintf("exec contract:%s, owner:%s occur error: %v", ctx.Contract, ctx.Owner, err))
	}
	return ret, err
}

func (w *WasmVmService) Register(funcName string, function interface{}) {
	w.registerFuncs[funcName] = function
}

func (w *WasmVmService) Start(node *node.Node) error {
	db, err := w.getDb()
	if err != nil {
		return errors.New("Economist fetch db service error")
	}
	w.db = db
	dgpWrap := table.NewSoGlobalWrap(w.db, &SINGLE_ID)
	if !dgpWrap.CheckExist() {
		return errors.New("the mainkey is already exist")
	}
	w.globalProps = dgpWrap.GetProps()
	return nil
}

func (w *WasmVmService) Stop() error {
	return nil
}
