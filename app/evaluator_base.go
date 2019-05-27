package app

import (
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm/injector"
	"github.com/sirupsen/logrus"
)

func opAssertE(err error, val string) {
	if err != nil {
		panic(val + " : " + err.Error())
	}
}

func opAssert(b bool, val string) {
	if !b {
		panic(val)
	}
}

func mustNoError(err error, val string) {
	if err != nil {
		panic(val + " : " + err.Error())
	}
}

type ApplyDelegate interface {
	Database() iservices.IDatabaseRW
	GlobalProp() iservices.IGlobalPropRW
	VMInjector() vminjector.Injector
	TrxObserver() iservices.ITrxObserver
	Logger() *logrus.Logger
}

type BaseDelegate struct {
	delegate ApplyDelegate
}

func (d *BaseDelegate) Database() iservices.IDatabaseRW {
	return d.delegate.Database()
}

func (d *BaseDelegate) GlobalProp() iservices.IGlobalPropRW {
	return d.delegate.GlobalProp()
}

func (d *BaseDelegate) VMInjector() vminjector.Injector {
	return d.delegate.VMInjector()
}

func (d *BaseDelegate) TrxObserver() iservices.ITrxObserver {
	return d.delegate.TrxObserver()
}

func (d *BaseDelegate) Logger() *logrus.Logger {
	return d.delegate.Logger()
}

type BaseEvaluator interface {
	Apply()
}

type EvaluatorCreator func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator

const sMetaKeyEvaluatorCreator = "op_meta_evaluator_creator"

func GetBaseEvaluator(delegate ApplyDelegate, op *prototype.Operation) BaseEvaluator {
	if value := prototype.GetGenericOperationMeta(op, sMetaKeyEvaluatorCreator); value != nil {
		evalCreator := value.(EvaluatorCreator)
		return evalCreator(delegate, prototype.GetBaseOperation(op))
	}
	panic("no matchable evaluator")
}

func RegisterEvaluator(opPtr interface{}, evalCreator EvaluatorCreator) {
	prototype.RegisterOperationMeta(opPtr, sMetaKeyEvaluatorCreator, evalCreator)
}
