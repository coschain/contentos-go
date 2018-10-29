package db

import (
	"testing"

	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/db/forkdb"
)

func TestForkDB(t *testing.T) {
	var p common.BlockID
	msb0 := &MockSignedBlock{
		Payload: []byte("hello0"),
		Num:     0,
		Prev:    p,
	}
	db := forkdb.NewDB()
	b := db.PushBlock(msb0)
	if b != msb0 {
		t.Error("wrong head")
	}

	prevID := msb0.Id()
	msb1 := &MockSignedBlock{
		Payload: []byte("hello1"),
		Num:     1,
		Prev:    prevID,
	}
	b = db.PushBlock(msb1)
	if b != msb1 {
		t.Error("wrong head")
	}

	prevID = msb1.Id()
	msb2 := &MockSignedBlock{
		Payload: []byte("hello2"),
		Num:     2,
		Prev:    prevID,
	}

	prevID = msb2.Id()
	msb3 := &MockSignedBlock{
		Payload: []byte("hello3"),
		Num:     3,
		Prev:    prevID,
	}

	b = db.PushBlock(msb3)
	if b != msb1 {
		t.Error("wrong head")
	}

	b = db.PushBlock(msb2)
	if b != msb2 {
		t.Error("wrong head")
	}

	b = db.PushBlock(msb3)
	if b != msb3 {
		t.Error("wrong head")
	}

	prevID = msb0.Id()
	msb1_1 := &MockSignedBlock{
		Payload: []byte("world1"),
		Num:     1,
		Prev:    prevID,
	}

	prevID = msb1_1.Id()
	msb1_2 := &MockSignedBlock{
		Payload: []byte("world2"),
		Num:     2,
		Prev:    prevID,
	}

	prevID = msb1_2.Id()
	msb1_3 := &MockSignedBlock{
		Payload: []byte("world3"),
		Num:     3,
		Prev:    prevID,
	}

	prevID = msb1_3.Id()
	msb1_4 := &MockSignedBlock{
		Payload: []byte("world4"),
		Num:     4,
		Prev:    prevID,
	}

	// prevID = msb1_4.Id()
	// msb1_5 := &MockSignedBlock{
	// 	Payload: []byte("world5"),
	// 	Num:     5,
	// 	Prev:    prevID,
	// }

	b = db.PushBlock(msb1_1)
	if b != msb3 {
		t.Error("wrong head")
	}
	b = db.PushBlock(msb1_3)
	if b != msb3 {
		t.Error("wrong head")
	}
	b = db.PushBlock(msb1_4)
	if b != msb3 {
		t.Error("wrong head")
	}
	b = db.PushBlock(msb1_2)
	if b != msb1_4 {
		t.Error("wrong head")
	}
}
