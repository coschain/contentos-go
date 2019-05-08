package storage

import (
	"fmt"
	"testing"
)

func BenchmarkIteration(b *testing.B) {
	db := NewMemoryDatabase()
	entries := 100000
	for i := 0; i < entries; i++ {
		s := fmt.Sprintf("%016d", i)
		_ = db.Put([]byte(s), []byte(s))
	}
	start, limit := fmt.Sprintf("%016d", entries * 4 / 10), fmt.Sprintf("%016d", entries * 6 / 10)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		db.Iterate([]byte(start), []byte(limit), false, func(key, value []byte) bool {
			return true
		})
	}
}