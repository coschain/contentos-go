package app

import (
	"fmt"
	"github.com/coschain/contentos-go/common/constants"
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
	Logger() *logrus.Logger
	HardFork() uint64
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

func (d *BaseDelegate) Logger() *logrus.Logger {
	return d.delegate.Logger()
}

func (d *BaseDelegate) HardFork() uint64 {
	return d.delegate.HardFork()
}

type BaseEvaluator interface {
	Apply()
}

type EvaluatorCreator func(delegate ApplyDelegate, op prototype.BaseOperation) BaseEvaluator

const sMetaKeyEvaluatorCreator = "op_meta_evaluator_creator"
const sMetaKeyEvaluatorHardFork = "op_meta_evaluator_hard_fork"

func GetBaseEvaluator(delegate ApplyDelegate, op *prototype.Operation, currentHardfork uint64) BaseEvaluator {
	if value := prototype.GetGenericOperationMeta(op, sMetaKeyEvaluatorHardFork); value != nil {
		minHardFork := value.(uint64)
		if currentHardfork < minHardFork {
			panic(fmt.Sprintf("evaluator only works after hard_fork = %d", minHardFork))
		}
	}
	if value := prototype.GetGenericOperationMeta(op, sMetaKeyEvaluatorCreator); value != nil {
		evalCreator := value.(EvaluatorCreator)
		return evalCreator(delegate, prototype.GetBaseOperation(op))
	}
	panic("no matchable evaluator")
}

func RegisterEvaluator(opPtr interface{}, evalCreator EvaluatorCreator) {
	RegisterEvaluatorWithMinHardFork(opPtr, evalCreator, constants.Original)
}

func RegisterEvaluatorWithMinHardFork(opPtr interface{}, evalCreator EvaluatorCreator, minHardFork uint64) {
	prototype.RegisterOperationMeta(opPtr, sMetaKeyEvaluatorCreator, evalCreator)
	prototype.RegisterOperationMeta(opPtr, sMetaKeyEvaluatorHardFork, minHardFork)
}
