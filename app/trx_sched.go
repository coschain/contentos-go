package app

import "github.com/coschain/contentos-go/prototype"

type ITrxScheduler interface {
	ScheduleTrxs(trxs []*prototype.SignedTransaction) [][]*prototype.SignedTransaction
}

type DefaultTrxScheduler struct{}

func (s DefaultTrxScheduler) ScheduleTrxs(trxs []*prototype.SignedTransaction) [][]*prototype.SignedTransaction {
	return [][]*prototype.SignedTransaction{ trxs }
}

type PropBasedTrxScheduler struct{}

func (s PropBasedTrxScheduler) ScheduleTrxs(trxs []*prototype.SignedTransaction) [][]*prototype.SignedTransaction {
	lines := [][]*prototype.SignedTransaction{ nil }
	props := make(map[string]int)
	possibleIndeps := make(map[int]map[string]bool)
	for i, tx := range trxs {
		p := make(map[string]bool)
		tx.GetAffectedProps(&p)
		dep := p["*"]
		if !dep {
			for k := range p {
				if props[k] > 0 {
					dep = true
				}
				props[k]++
			}
		}
		if dep {
			lines[0] = append(lines[0], tx)
		} else {
			possibleIndeps[i] = p
		}
	}
	for i, p := range possibleIndeps {
		s := 0
		for k := range p {
			s += props[k]
		}
		if s == len(p) {
			lines[0] = append(lines[0], trxs[i])
		} else {
			lines = append(lines, []*prototype.SignedTransaction{ trxs[i] })
		}
	}
	return lines
}
