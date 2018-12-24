package vm

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm/context"
	"github.com/sirupsen/logrus"
)

var (
	// fixme: the single id should be share with service
	SINGLE_ID int32 = 1
)

type WasmVmService struct {
	ctx             *node.ServiceContext
	registerFuncs   map[string]interface{}
	registerFuncGas map[string]uint64

	logger      *logrus.Logger
	db          iservices.IDatabaseService
	globalProps *prototype.DynamicProperties
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
	return &WasmVmService{ctx: ctx,
		registerFuncs:   make(map[string]interface{}),
		registerFuncGas: make(map[string]uint64),
		logger:          logrus.New()}, nil
}

func (w *WasmVmService) Run(ctx *vmcontext.Context) (uint32, error) {
	cosVM := NewCosVM(ctx, w.db, w.globalProps, w.logger)
	for funcName, function := range w.registerFuncs {
		cosVM.Register(funcName, function, w.registerFuncGas[funcName])
	}
	ret, err := cosVM.Run()
	if err != nil {
		w.logger.Error(fmt.Sprintf("exec contract:%s, owner:%s occur error: %v", ctx.Contract, ctx.Owner, err))
	}
	return ret, err
}

func (w *WasmVmService) Register(funcName string, function interface{}, gas uint64) {
	w.registerFuncs[funcName] = function
	w.registerFuncGas[funcName] = gas
}

func (w *WasmVmService) Validate(ctx *vmcontext.Context) error {
	cosVM := NewCosVM(ctx, w.db, w.globalProps, w.logger)
	for funcName, function := range w.registerFuncs {
		cosVM.Register(funcName, function, w.registerFuncGas[funcName])
	}
	err := cosVM.Validate()
	if err != nil {
		w.logger.Error(fmt.Sprintf("validate contract:%s, owner:%s occur error: %v", ctx.Contract, ctx.Owner, err))
	}
	return err
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
