package app

import (
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/vm/injector"
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
	trxCtx	   *TrxContext
}

type BaseEvaluator interface {
	Apply()
}

func GetBaseEvaluator(ctx *ApplyContext, op *prototype.Operation) BaseEvaluator {
	switch op.Op.(type) {
	case *prototype.Operation_Op1:
		eva := &AccountCreateEvaluator{ctx: ctx, op: op.GetOp1()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op2:
		eva := &TransferEvaluator{ctx: ctx, op: op.GetOp2()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op3:
		eva := &BpRegisterEvaluator{ctx: ctx, op: op.GetOp3()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op4:
		eva := &BpUnregisterEvaluator{ctx: ctx, op: op.GetOp4()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op5:
		eva := &BpVoteEvaluator{ctx: ctx, op: op.GetOp5()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op6:
		eva := &PostEvaluator{ctx: ctx, op: op.GetOp6()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op7:
		eva := &ReplyEvaluator{ctx: ctx, op: op.GetOp7()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op8:
		eva := &FollowEvaluator{ctx: ctx, op: op.GetOp8()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op9:
		eva := &VoteEvaluator{ctx: ctx, op: op.GetOp9()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op10:
		eva := &TransferToVestingEvaluator{ctx: ctx, op: op.GetOp10()}
		return BaseEvaluator(eva)
	//case *prototype.Operation_Op11:
	//	eva := &ClaimEvaluator{ctx: ctx, op: op.GetOp11()}
	//	return BaseEvaluator(eva)
	//case *prototype.Operation_Op12:
	//	eva := &ClaimAllEvaluator{ctx: ctx, op: op.GetOp12()}
	//	return BaseEvaluator(eva)
	case *prototype.Operation_Op13:
		eva := &ContractDeployEvaluator{ctx: ctx, op: op.GetOp13()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op14:
		eva := &ContractApplyEvaluator{ctx: ctx, op: op.GetOp14()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op15:
		eva := &ReportEvaluator{ctx: ctx, op: op.GetOp15()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op16:
		eva := &ConvertVestingEvaluator{ctx: ctx, op: op.GetOp16()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op17:
		eva := &StakeEvaluator{ctx: ctx, op: op.GetOp17()}
		return BaseEvaluator(eva)
	case *prototype.Operation_Op18:
		eva := &UnStakeEvaluator{ctx: ctx, op: op.GetOp18()}
		return BaseEvaluator(eva)
	default:
		panic("no matchable evaluator")
	}
}
