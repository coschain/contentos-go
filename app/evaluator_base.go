package app

import "github.com/coschain/contentos-go/iservices"

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

type ApplyContext struct {
	db      iservices.IDatabaseService
	control iservices.ITrxPool
	trxCtx *TrxContext
}

type BaseEvaluator interface {
	Apply()
}
