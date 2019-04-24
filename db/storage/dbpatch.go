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

func (p *DatabasePatch) Iterate(start, limit []byte, reverse bool, callback func(key, value []byte) bool) {
	p.s.Iterate(start, limit, reverse, callback)
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

func (p *DatabasePatch) NewPatch() iservices.IDatabasePatch {
	return NewDatabasePatch(p.s)
}
