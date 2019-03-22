package app

import "github.com/coschain/contentos-go/prototype"

// ITrxScheduler is a scheduler for multiple transactions.
// Its purpose is to split an incoming transaction group into multiple independent sub-groups.
// Sub-groups are safe for concurrent application while transactions of the same sub-group
// are dependent and must be applied in order.
type ITrxScheduler interface {
	ScheduleTrxWrappers(trxs []*prototype.TransactionWrapper) [][]*prototype.TransactionWrapper
	ScheduleTrxEstResults(trxs []*prototype.EstimateTrxResult) [][]*prototype.EstimateTrxResult
}

// DefaultTrxScheduler schedules nothing.
// It just outputs a single sub-group with all incoming transactions.
type DefaultTrxScheduler struct{}

func (s DefaultTrxScheduler) ScheduleTrxWrappers(trxs []*prototype.TransactionWrapper) [][]*prototype.TransactionWrapper {
	return [][]*prototype.TransactionWrapper{ trxs }
}

func (s DefaultTrxScheduler) ScheduleTrxEstResults(trxs []*prototype.EstimateTrxResult) [][]*prototype.EstimateTrxResult {
	return [][]*prototype.EstimateTrxResult{ trxs }
}


// PropBasedTrxScheduler split sub-groups based on affected properties of each transaction.
type PropBasedTrxScheduler struct{}

func (s PropBasedTrxScheduler) ScheduleTrxWrappers(trxs []*prototype.TransactionWrapper) [][]*prototype.TransactionWrapper {
	groups := s.schedule(len(trxs), func(idx int) *prototype.SignedTransaction {
		return trxs[idx].SigTrx
	})
	if len(groups) <= 1 {
		return [][]*prototype.TransactionWrapper{trxs}
	}
	g := make([][]*prototype.TransactionWrapper, len(groups))
	for i := range g {
		a := groups[i]
		b := make([]*prototype.TransactionWrapper, len(a))
		for j, k := range a {
			b[j] = trxs[k]
		}
		g[i] = b
	}
	return g
}

func (s PropBasedTrxScheduler) ScheduleTrxEstResults(trxs []*prototype.EstimateTrxResult) [][]*prototype.EstimateTrxResult {
	groups := s.schedule(len(trxs), func(idx int) *prototype.SignedTransaction {
		return trxs[idx].SigTrx
	})
	if len(groups) <= 1 {
		return [][]*prototype.EstimateTrxResult{trxs}
	}
	g := make([][]*prototype.EstimateTrxResult, len(groups))
	for i := range g {
		a := groups[i]
		b := make([]*prototype.EstimateTrxResult, len(a))
		for j, k := range a {
			b[j] = trxs[k]
		}
		g[i] = b
	}
	return g
}

func (s PropBasedTrxScheduler) schedule(count int, trxGetter func(idx int)*prototype.SignedTransaction) [][]int {
	groups := [][]int{nil}
	props := make(map[string]int)
	possibleIndeps := make(map[int]map[string]bool)

	// traverse all transactions and collect their affected props.
	for i := 0; i < count; i++ {
		p := make(map[string]bool)
		trxGetter(i).GetAffectedProps(&p)

		// if a transaction affects everything, concurrency is impossible.
		if p["*"] {
			return nil
		}

		// update number of changers of each prop
		dep := false
		for k := range p {
			if props[k] > 0 {
				// the prop is affected by current trx and some other trxs,
				// so current trx is not independent for sure.
				dep = true
			}
			props[k]++
		}
		if dep {
			// current trx is not independent, put it to sub-group #0.
			groups[0] = append(groups[0], i)
		} else {
			// remember this possible independent trx
			possibleIndeps[i] = p
		}
	}
	// recheck possible independent trxs to see if they're really independent.
	for i, p := range possibleIndeps {
		// a trx is independent iff each of its affected prop has only 1 changer.
		s := 0
		for k := range p {
			s += props[k]
		}
		if s > len(p) {
			groups[0] = append(groups[0], i)
		} else {
			groups = append(groups, []int{i})
		}
	}
	// if we got only 1 sub-group, just return nil, which means on concurrency.
	if len(groups) == 1 {
		groups = nil
	}
	return groups
}
