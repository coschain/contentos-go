package app

import (
	"fmt"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/prototype"
	"sync"
)

// MultiTrxsApplier concurrently applies multiple transactions.
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
	// split incoming trxs into independent sub-groups.
	g := a.sched.ScheduleTrxEstResults(trxs)

	// apply each group concurrently
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

// applyGroup applies transaction of given group one by one.
func (a *MultiTrxsApplier) applyGroup(group []*prototype.EstimateTrxResult) {
	// first, set up a database patch to save all changes
	groupDb := a.db.NewPatch()
	for _, trx := range group {
		// one more database layer for transaction
		txDb := groupDb.NewPatch()
		// apply the transaction on transaction db layer
		err := a.applySingle(txDb, trx)
		// commit transaction changes if no errors
		if err == nil {
			err = txDb.Apply()
		}
	}
	// finally, commit the changes
	groupDb.Apply()
}

func (a *MultiTrxsApplier) applySingle(db iservices.IDatabaseRW, trx *prototype.EstimateTrxResult) (err error) {
	defer func() {
		// recover from panic and return an error
		if e := recover(); e != nil {
			err = fmt.Errorf("%v", e)
		}
	}()
	// singleApplier is not panic-free
	a.singleApplier(db, trx)
	return
}
