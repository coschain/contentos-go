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

// TODO replace applyContext to TrxContext
type ApplyContext struct {
	db         iservices.IDatabaseRW
	control    iservices.IGlobalPropRW
	vmInjector vminjector.Injector
	observer iservices.ITrxObserver
	log     *logrus.Logger
}

type BaseEvaluator interface {
	Apply()
}

type EvaluatorCreator func(ctx *ApplyContext, op prototype.BaseOperation) BaseEvaluator

const sMetaKeyEvaluatorCreator = "op_meta_evaluator_creator"

func GetBaseEvaluator(ctx *ApplyContext, op *prototype.Operation) BaseEvaluator {
	if value := prototype.GetGenericOperationMeta(op, sMetaKeyEvaluatorCreator); value != nil {
		evalCreator := value.(EvaluatorCreator)
		return evalCreator(ctx, prototype.GetBaseOperation(op))
	}
	panic("no matchable evaluator")
}

func RegisterEvaluator(opPtr interface{}, evalCreator EvaluatorCreator) {
	prototype.RegisterOperationMeta(opPtr, sMetaKeyEvaluatorCreator, evalCreator)
}
