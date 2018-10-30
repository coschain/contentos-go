package storage

import (
	"testing"
	"bytes"
	"math/rand"
	"io/ioutil"
	"os"
	"path/filepath"
)

func requireSuccessGet(t *testing.T, db Database, key []byte, correctValue []byte) {
	value, err := db.Get(key)
	if err != nil {
		t.Fatalf("Failed Get key=%v, err: %v", string(key), err)
	}
	if bytes.Compare(value, correctValue) != 0 {
		t.Fatalf("Error value for key=%v. got %v, expecting %v", string(key), string(value), string(correctValue))
	}
}

func requireErrorGet(t *testing.T, db Database, key []byte) {
	_, err := db.Get(key)
	if err == nil {
		t.Fatalf("Get non-existent key=%v", string(key))
	}
}

func requireSuccessPut(t *testing.T, db Database, key []byte, value []byte) {
	err := db.Put(key, value)
	if err != nil {
		t.Fatalf("Failed Put key=%v, value=%v, err %v", string(key), string(value), err)
	}
}

func requireSuccessDel(t *testing.T, db Database, key []byte) {
	err := db.Delete(key)
	if err != nil {
		t.Fatalf("Failed Delete key=%v, err %v", string(key), err)
	}
}

func requireIteratorKeyValue(t *testing.T, it Iterator, key []byte, value []byte) {
	data, err := it.Key()
	if err != nil {
		t.Fatalf("Failed Iterator.Key, key=%v, err: %v", string(key), err)
	}
	if bytes.Compare(data, key) != 0 {
		t.Fatalf("Incorrect Iterator Key, got %v, expecting %v", string(data), string(key))
	}
	data, err = it.Value()
	if err != nil {
		t.Fatalf("Failed Iterator.Value, key=%v, err: %v", string(key), err)
	}
	if bytes.Compare(data, value) != 0 {
		t.Fatalf("Incorrect Iterator Value, got %v, expecting %v", string(data), string(value))
	}
}

func dbTest(t *testing.T, db Database) {

	// fail to get non-existent keys
	requireErrorGet(t, db, []byte("key_one"))
	requireErrorGet(t, db, []byte("key_two"))
	requireErrorGet(t, db, []byte("key_three"))

	// normal puts
	requireSuccessPut(t, db, []byte("key_one"), []byte("value_one"))
	requireSuccessPut(t, db, []byte("key_two"), []byte("value_two"))
	requireSuccessPut(t, db, []byte("key_three"), []byte("value_three"))

	// fetched values must be the same as put values
	requireSuccessGet(t, db, []byte("key_one"), []byte("value_one"))
	requireSuccessGet(t, db, []byte("key_two"), []byte("value_two"))
	requireSuccessGet(t, db, []byte("key_three"), []byte("value_three"))

	// delete existent keys
	requireSuccessDel(t, db, []byte("key_two"))

	// it's ok to return nil error when deleting non-existent keys
	requireSuccessDel(t, db, []byte("key_ten"))

	// key_two was deleted, cannot get it
	requireErrorGet(t, db, []byte("key_two"))

	// key_three is still available
	requireSuccessGet(t, db, []byte("key_three"), []byte("value_three"))

	// puts 2 more k-v
	requireSuccessPut(t, db, []byte("key_four"), []byte("value_four"))
	requireSuccessPut(t, db, []byte("key_five"), []byte("value_five"))

	// range scan for key_five, key_four, key_one. key_three is filtered by limit "key_s"
	it := db.NewIterator([]byte("key_"), []byte("key_s"))
	it.Next()
	requireIteratorKeyValue(t, it, []byte("key_five"), []byte("value_five"))
	it.Next()
	requireIteratorKeyValue(t, it, []byte("key_four"), []byte("value_four"))
	it.Next()
	requireIteratorKeyValue(t, it, []byte("key_one"), []byte("value_one"))
	db.DeleteIterator(it)

	// batch of deletions and puts
	b := db.NewBatch()
	b.Delete([]byte("key_one"))
	b.Delete([]byte("key_three"))
	b.Delete([]byte("key_five"))
	b.Delete([]byte("key_four"))
	b.Put([]byte("key_two"), []byte("2"))
	b.Write()
	db.DeleteBatch(b)

	// test what's left by the batch
	requireSuccessGet(t, db, []byte("key_two"), []byte("2"))
	requireErrorGet(t, db, []byte("key_four"))
	requireErrorGet(t, db, []byte("key_five"))
}

func TestMemoryDatabase(t *testing.T) {
	db := NewMemoryDatabase()
	defer db.Close()

	dbTest(t, db)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func randomString(size uint) string {
	b := make([]byte, size)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func TestLevelDatabase(t *testing.T) {
	dir, err := ioutil.TempDir("", "lvldb")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	fn := filepath.Join(dir, randomString(8))
	db, err := NewLevelDatabase(fn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	dbTest(t, db)
}
