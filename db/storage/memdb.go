package storage

func NewMemoryDatabase() Database {
	return NewRedblackDatabase()
}
