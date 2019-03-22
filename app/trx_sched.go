package app

import "github.com/coschain/contentos-go/prototype"

type ITrxScheduler interface {
	ScheduleTrxWrappers(trxs []*prototype.TransactionWrapper) [][]*prototype.TransactionWrapper
	ScheduleTrxEstResults(trxs []*prototype.EstimateTrxResult) [][]*prototype.EstimateTrxResult
}

type DefaultTrxScheduler struct{}

func (s DefaultTrxScheduler) ScheduleTrxWrappers(trxs []*prototype.TransactionWrapper) [][]*prototype.TransactionWrapper {
	return [][]*prototype.TransactionWrapper{ trxs }
}

func (s DefaultTrxScheduler) ScheduleTrxEstResults(trxs []*prototype.EstimateTrxResult) [][]*prototype.EstimateTrxResult {
	return [][]*prototype.EstimateTrxResult{ trxs }
}

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
	for i := 0; i < count; i++ {
		p := make(map[string]bool)
		trxGetter(i).GetAffectedProps(&p)
		if p["*"] {
			return nil
		}
		dep := false
		for k := range p {
			if props[k] > 0 {
				dep = true
			}
			props[k]++
		}
		if dep {
			groups[0] = append(groups[0], i)
		} else {
			possibleIndeps[i] = p
		}
	}
	for i, p := range possibleIndeps {
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
	if len(groups) == 1 {
		groups = nil
	}
	return groups
}
