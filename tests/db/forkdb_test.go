package db

import (
	"fmt"
	"runtime"
	"testing"

	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/db/forkdb"
)

func requireEqual(t *testing.T, a interface{}, b interface{}) {
	_, _, line, _ := runtime.Caller(1)
	errStr := fmt.Sprintf("requireEqual: %d", line)
	if a != b {
		t.Error(errStr)
	}
}

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

	b = db.PushBlock(msb1)
	requireEqual(t, b, msb1)
	requireEqual(t, 2, db.TotalBlockNum())

	b = db.PushBlock(msb3)
	requireEqual(t, b, msb1)
	requireEqual(t, 2, db.TotalBlockNum())

	b = db.PushBlock(msb2)
	requireEqual(t, b, msb2)
	requireEqual(t, 3, db.TotalBlockNum())

	b = db.PushBlock(msb3)
	requireEqual(t, b, msb3)
	requireEqual(t, 4, db.TotalBlockNum())

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
	requireEqual(t, b, msb3)
	requireEqual(t, 5, db.TotalBlockNum())

	b = db.PushBlock(msb1_3)
	requireEqual(t, b, msb3)
	requireEqual(t, 5, db.TotalBlockNum())

	b = db.PushBlock(msb1_4)
	requireEqual(t, b, msb3)
	requireEqual(t, 5, db.TotalBlockNum())

	b = db.PushBlock(msb1_2)
	requireEqual(t, b, msb1_4)
	requireEqual(t, 8, db.TotalBlockNum())

	var fullBranch [2][]common.BlockID
	fullBranch[0] = append(fullBranch[0], msb3.Id(), msb2.Id(), msb1.Id(), msb0.Id())
	fullBranch[1] = append(fullBranch[1], msb1_4.Id(), msb1_3.Id(), msb1_2.Id(), msb1_1.Id(), msb0.Id())
	fetchedBranch, err := db.FetchBranch(msb3.Id(), msb1_4.Id())
	if err != nil {
		t.Error("FetchBranch failed")
	}
	requireEqual(t, len(fullBranch[0]), len(fetchedBranch[0]))
	requireEqual(t, len(fullBranch[1]), len(fetchedBranch[1]))
	for i := 0; i < len(fullBranch[0]); i++ {
		requireEqual(t, fullBranch[0][i], fetchedBranch[0][i])
		//fmt.Printf("fullBranch[0][%d] = %x, fetchedBranch[0][%d] = %x\n", i, fullBranch[0][i], i, fetchedBranch[0][i])
	}
	for i := 0; i < len(fullBranch[1]); i++ {
		requireEqual(t, fullBranch[1][i], fetchedBranch[1][i])
	}

	fetchedBranch, err = db.FetchBranch(msb0.Id(), msb1_3.Id())
	requireEqual(t, 1, len(fetchedBranch[0]))
	requireEqual(t, 4, len(fetchedBranch[1]))
	for i := 0; i < 1; i++ {
		requireEqual(t, msb0.Id(), fetchedBranch[0][i])
	}
	for i := 0; i < 4; i++ {
		requireEqual(t, fullBranch[1][i+1], fetchedBranch[1][i])
	}

	prevID = msb1_2.Id()
	msb2_3 := &MockSignedBlock{
		Payload: []byte("knock3"),
		Num:     3,
		Prev:    prevID,
	}
	b = db.PushBlock(msb2_3)
	requireEqual(t, b, msb1_4)
	requireEqual(t, 9, db.TotalBlockNum())

	var afterCommit []common.BlockID
	afterCommit = append(afterCommit, msb1_2.Id(), msb1_3.Id(), msb1_4.Id())
	db.Commit(msb1_2.Id())
	_, ids, err := db.FetchBlocksSince(msb1_2.Id())
	for i := 0; i < len(ids); i++ {
		requireEqual(t, afterCommit[i], ids[i])
	}
	requireEqual(t, 3, db.TotalBlockNum())
}
