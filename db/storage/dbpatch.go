package storage

import "github.com/coschain/contentos-go/iservices"

type DatabasePatch struct {
	s *dbSession
}

func NewDatabasePatch(db Database) *DatabasePatch {
	return &DatabasePatch{
		s: &dbSession{
			db: db,
			mem: NewMemoryDatabase(),
			removals: make(map[string]bool),
		},
	}
}

func (p *DatabasePatch) Has(key []byte) (bool, error) {
	return p.s.Has(key)
}

func (p *DatabasePatch) Get(key []byte) ([]byte, error) {
	return p.s.Get(key)
}

func (p *DatabasePatch) Put(key []byte, value []byte) error {
	return p.s.Put(key, value)
}

func (p *DatabasePatch) Delete(key []byte) error {
	return p.s.Delete(key)
}

func (p *DatabasePatch) NewIterator(start []byte, limit []byte) iservices.IDatabaseIterator {
	return p.s.NewIterator(start, limit)
}

// same as NewIterator, but iteration will be in reversed order.
func (p *DatabasePatch) NewReversedIterator(start []byte, limit []byte) iservices.IDatabaseIterator {
	return p.s.NewReversedIterator(start, limit)
}

func (p *DatabasePatch) DeleteIterator(it iservices.IDatabaseIterator) {
	p.s.DeleteIterator(it)
}

func (p *DatabasePatch) NewBatch() iservices.IDatabaseBatch {
	return p.s.NewBatch()
}

func (p *DatabasePatch) DeleteBatch(b iservices.IDatabaseBatch) {
	p.s.DeleteBatch(b)
}

func (p *DatabasePatch) Apply() error {
	return p.s.commit()
}
