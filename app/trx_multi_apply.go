package app

import (
	"fmt"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"sync"
)

type MultiTrxsApplier struct {
	db            iservices.IDatabaseService
	singleApplier func(db iservices.IDatabaseRW, trx *prototype.EstimateTrxResult)
	sched         ITrxScheduler
}

func NewMultiTrxsApplier(db iservices.IDatabaseService, singleApplier func(iservices.IDatabaseRW, *prototype.EstimateTrxResult)) *MultiTrxsApplier {
	return &MultiTrxsApplier{
		db: db,
		singleApplier: singleApplier,
		sched: DefaultTrxScheduler{},
	}
}

func (a *MultiTrxsApplier) Apply(trxs []*prototype.EstimateTrxResult) {
	g := a.sched.ScheduleTrxEstResults(trxs)
	var wg sync.WaitGroup
	wg.Add(len(g))
	for i := range g {
		go func(idx int) {
			defer wg.Done()
			a.applyGroup(g[idx])
		}(i)
	}
	wg.Wait()
}

func (a *MultiTrxsApplier) applyGroup(group []*prototype.EstimateTrxResult) {
	groupDb := a.db.NewPatch()
	for _, trx := range group {
		txDb := groupDb.NewPatch()
		err := a.applySingle(txDb, trx)
		if err == nil {
			err = txDb.Apply()
		}
	}
	groupDb.Apply()
}

func (a *MultiTrxsApplier) applySingle(db iservices.IDatabaseRW, trx *prototype.EstimateTrxResult) (err error) {
	defer func() {
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	a.singleApplier(db, trx)
	return
}
