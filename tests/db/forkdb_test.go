package db

import (
	"testing"

	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/db/forkdb"
	"github.com/stretchr/testify/assert"
)

func TestForkDB(t *testing.T) {
	assert := assert.New(t)
	var p common.BlockID
	msb0 := &MockSignedBlock{
		Payload: []byte("hello0"),
		Num:     0,
		Prev:    p,
	}
	db := forkdb.NewDB()
	assert.Equal(db.Empty(), true)
	db.PushBlock(msb0)
	assert.Equal(db.Head(), msb0, "wrong head")
	assert.Equal(db.Empty(), false)

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

	db.PushBlock(msb1)
	assert.Equal(db.Head(), msb1, "wrong head")
	assert.Equal(2, db.TotalBlockNum(), "wrong total block number")

	db.PushBlock(msb3)
	assert.Equal(db.Head(), msb1, "wrong head")
	assert.Equal(2, db.TotalBlockNum(), "wrong total block number")

	db.PushBlock(msb2)
	assert.Equal(db.Head(), msb2, "wrong head")
	assert.Equal(3, db.TotalBlockNum(), "wrong total block number")

	db.PushBlock(msb3)
	assert.Equal(db.Head(), msb3, "wrong head")
	assert.Equal(4, db.TotalBlockNum(), "wrong total block number")

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

	db.PushBlock(msb1_1)
	assert.Equal(db.Head(), msb3, "wrong head")
	assert.Equal(5, db.TotalBlockNum(), "wrong total block number")

	db.PushBlock(msb1_3)
	assert.Equal(db.Head(), msb3, "wrong head")
	assert.Equal(5, db.TotalBlockNum(), "wrong total block number")

	db.PushBlock(msb1_4)
	assert.Equal(db.Head(), msb3, "wrong head")
	assert.Equal(5, db.TotalBlockNum(), "wrong total block number")

	db.PushBlock(msb1_2)
	assert.Equal(db.Head(), msb1_4, "wrong head")
	assert.Equal(8, db.TotalBlockNum(), "wrong total block number")

	var fullBranch [2][]common.BlockID
	fullBranch[0] = append(fullBranch[0], msb3.Id(), msb2.Id(), msb1.Id(), msb0.Id())
	fullBranch[1] = append(fullBranch[1], msb1_4.Id(), msb1_3.Id(), msb1_2.Id(), msb1_1.Id(), msb0.Id())
	fetchedBranch, err := db.FetchBranch(msb3.Id(), msb1_4.Id())
	if err != nil {
		t.Error("FetchBranch failed")
	}
	assert.Equal(len(fullBranch[0]), len(fetchedBranch[0]), "len of branch0 mismatch")
	assert.Equal(len(fullBranch[1]), len(fetchedBranch[1]), "len of branch1 mismatch")
	for i := 0; i < len(fullBranch[0]); i++ {
		assert.Equal(fullBranch[0][i], fetchedBranch[0][i], "block id mismatch")
		//fmt.Printf("fullBranch[0][%d] = %x, fetchedBranch[0][%d] = %x\n", i, fullBranch[0][i], i, fetchedBranch[0][i])
	}
	for i := 0; i < len(fullBranch[1]); i++ {
		assert.Equal(fullBranch[1][i], fetchedBranch[1][i], "block id mismatch")
	}

	fetchedBranch, err = db.FetchBranch(msb0.Id(), msb1_3.Id())
	assert.Equal(1, len(fetchedBranch[0]), "len of branch0 mismatch")
	assert.Equal(4, len(fetchedBranch[1]), "len of branch1 mismatch")
	for i := 0; i < 1; i++ {
		assert.Equal(msb0.Id(), fetchedBranch[0][i])
	}
	for i := 0; i < 4; i++ {
		assert.Equal(fullBranch[1][i+1], fetchedBranch[1][i])
	}

	prevID = msb1_2.Id()
	msb2_3 := &MockSignedBlock{
		Payload: []byte("knock3"),
		Num:     3,
		Prev:    prevID,
	}
	db.PushBlock(msb2_3)
	assert.Equal(db.Head(), msb1_4)
	assert.Equal(9, db.TotalBlockNum())

	var afterCommit []common.BlockID
	afterCommit = append(afterCommit, msb1_3.Id(), msb1_4.Id())
	db.Commit(msb1_2.Id())
	_, ids, err := db.FetchBlocksSince(msb1_2.Id())
	for i := 0; i < len(ids); i++ {
		assert.Equal(afterCommit[i], ids[i])
	}
	assert.Equal(4, db.TotalBlockNum())
}
