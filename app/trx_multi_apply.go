package app

import (
	"fmt"
	"github.com/coschain/contentos-go/iservices"
	"sync"
)

type SingleTrxApplier func(iservices.IDatabaseRW, *TrxEntry)

// MultiTrxsApplier concurrently applies multiple transactions.
type MultiTrxsApplier struct {
	db            iservices.IDatabaseService
	singleApplier SingleTrxApplier
	sched         ITrxScheduler
}

func NewMultiTrxsApplier(db iservices.IDatabaseService, singleApplier SingleTrxApplier) *MultiTrxsApplier {
	return &MultiTrxsApplier{
		db: db,
		singleApplier: singleApplier,
		sched: PropBasedTrxScheduler{},
	}
}

func (a *MultiTrxsApplier) Apply(trxs []*TrxEntry) {
	// split incoming trxs into independent sub-groups.
	g := a.sched.ScheduleTrxs(trxs)

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
func (a *MultiTrxsApplier) applyGroup(group []*TrxEntry) {
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
	_ = groupDb.Apply()
}

func (a *MultiTrxsApplier) applySingle(db iservices.IDatabaseRW, trx *TrxEntry) (err error) {
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
