package storage

type dbIterator struct {

}

func NewMergedIterator(databases []Database) *dbIterator {
	return nil
}

func NewPatchedIterator(patch, base Database, patchDeletes map[string]bool) *dbIterator {
	return nil
}

func (it *dbIterator) Iterate(start, limit []byte, reverse bool, callback func(key, value []byte) bool) {

}
