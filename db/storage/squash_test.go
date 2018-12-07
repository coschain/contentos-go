package storage

import (
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/common/encoding/kope"
	"os"
	"testing"
)

var fPath = "squash.db"
var sdb *SquashableDatabase
var db *LevelDatabase
var bCnt = 1000
var trxCnt = 100

func BenchmarkGenerateBlock(b *testing.B)  {
	os.RemoveAll(fPath)
	if sdb == nil {
		tdb,err := NewLevelDatabase(fPath)
		db = tdb
		if err == nil && db != nil {
			sdb = NewSquashableDatabase(db,true)
		}else{
			b.Fatalf("create level db fail")
		}
	}
	b.ResetTimer()
	for k := 0 ;  k < b.N ; k++ {
		if k + 1 < b.N  {
			return
		}
		//if sdb.TransactionHeight() > 0 {
		//	return
		//}
		key1 := 1
		for i := 1; i < bCnt + 1 ; i++ {
			tag := fmt.Sprintf("block%d",i)
			sdb.BeginTransactionWithTag(tag)
			//fmt.Printf("the block height is %d \n",sdb.TransactionHeight())
			for j := 1; j < trxCnt + 1 ; j++ {
				sdb.BeginTransaction()
				key,err := kope.Encode(key1)
				if err != nil {
					b.Fatalf("encode the key %d fail",key1)
				}
				val := []byte(fmt.Sprintf("block%d_val%d",i,j))
				putValue(b,sdb,key,val)
				err = sdb.EndTransaction(true)
				if err != nil {
					b.Fatalf("Faild to EndTransaction")
				}
				key1 ++
			}
			//fmt.Printf("the trans height is %d \n",sdb.TransactionHeight())
		}
		tag := fmt.Sprintf("block%d",bCnt)
		err := sdb.Squash(tag)
		if err != nil {
			b.Fatalf("squash fail")
		}
	}
}

func BenchmarkPushBlock(b *testing.B) {
	if sdb != nil && sdb.TransactionHeight() > 0 {
		return
	}
	os.RemoveAll(fPath)
    if sdb == nil {
    	 tdb,err := NewLevelDatabase(fPath)
    	 db = tdb
    	 if err == nil && db != nil {
			 sdb = NewSquashableDatabase(db,true)
		 }else{
		 	b.Fatalf("create level db fail")
		 }
	}
	 b.ResetTimer()
	 for k := 0 ;  k < b.N ; k++ {
		 if sdb.TransactionHeight() > 0 {
			return
		 }
		 var key2 = bCnt*trxCnt + 1
		 for i := bCnt+1; i < 2*bCnt + 1 ; i++ {
			 tag := fmt.Sprintf("block%d",i)
			 sdb.BeginTransactionWithTag(tag)
			 //fmt.Printf("the block height is %d \n",sdb.TransactionHeight())
			 for j := 1; j < trxCnt + 1 ; j++ {
				sdb.BeginTransaction()
				key,err := kope.Encode(key2)
				if err != nil {
					b.Fatalf("encode the key %d fail",key2)
				}
				val := []byte(fmt.Sprintf("block%d_val%d",i,j))
				putValue(b,sdb,key,val)
				err = sdb.EndTransaction(true)
				if err != nil {
					b.Fatalf("Faild to EndTransaction")
				}
				 key2 ++
			 }
			 //fmt.Printf("the trans height is %d \n",sdb.TransactionHeight())
		 }
	 }
}

func BenchmarkQuery(b *testing.B) {
	 b.ResetTimer()
	 for i := 0 ; i < b.N ; i++ {
		 err := queryVal(b, sdb, bCnt*trxCnt)
		 if err != nil {
			 b.Fatalf("get the value of key=%d fail,error is %s",bCnt*trxCnt,err)
		 }

		 //err = queryVal(b, sdb, 20)
		 //if err != nil {
			// b.Fatalf("get the value of key=10 fail,error is %s",err)
		 //}
		 //
		 //
		 //k := bCnt*10+trxCnt
		 //err = queryVal(b, sdb, k)
		 //if err != nil {
			// b.Fatalf("get the value of key=%d fail,error is %s", k, err)
		 //}
	 }

}


func BenchmarkIterator(b *testing.B)  {
	b.ResetTimer()
	for i := 0 ; i < b.N ; i++ {
		start,err := kope.Encode(1)
		if err != nil {
			b.Fatalf("encode the start key fail")
		}
		iter := sdb.NewIterator(start, nil)
		for iter.Next() {
			_,err := iter.Key()
			if err != nil {
				b.Fatalf("get the key in iterator fail")
			}
			_,err = iter.Value()
			if err != nil {
				b.Fatalf("get the value in iterator fail")
			}
			//fmt.Printf("the value is %s \n",string(val))
		}
	}

}

func BenchmarkRevIterator(b *testing.B)  {
	b.ResetTimer()
	for i := 0 ; i < b.N ; i++ {
		start,err := kope.Encode(bCnt*trxCnt)
		if err != nil {
			b.Fatalf("encode the start key fail")
		}
		iter := sdb.NewIterator(start, nil)
		for iter.Next() {
			_,err := iter.Key()
			if err != nil {
				b.Fatalf("get the key in iterator fail")
			}
			_,err = iter.Value()
			if err != nil {
				b.Fatalf("get the value in iterator fail")
			}
			//fmt.Printf("the value is %s \n",string(val))
		}
	}
}

//func TestEndTrx(b *testing.B)  {
//	for i := 0 ; i < 100 ; i++ {
//		err := sdb.EndTransaction(true)
//		if err != nil {
//			b.Fatalf("End transaction fail,the error is %s",err)
//		}
//	}
//}


func BenchmarkModify(b *testing.B)  {
	b.ResetTimer()
	for i := 0 ; i < b.N ; i++ {
		key,_ := kope.Encode(1)
		err := mdVal(b,sdb,key,[]byte("block1_1"))
		if err != nil {
			b.Fatalf("modify the key %s fail","blcok1_key1")
		}

	}

}

func BenchmarkDelVal(b *testing.B)  {
	b.ResetTimer()
	for k := 0 ; k < b.N ; k ++ {
		key,err := kope.Encode(bCnt*trxCnt)
		err = deleteVal(b, sdb , key)
		if err != nil {
			b.Fatalf("delete the key %s fail \n", string(key))
		}
	}
}

//func BenchmarkSquash(b *testing.B) {
//	b.ResetTimer()
//	for i := 0 ;  i < b.N ; i++ {
//		if sdb.TransactionHeight() < 1 {
//			return
//		}
//		tag := fmt.Sprintf("block%d",bCnt)
//		err := sdb.Squash(tag)
//		if err != nil {
//			b.Fatalf("squash the tag %s fail,error is %s",tag,err)
//		}
//	}
//	defer db.Close()
//    defer sdb.Close()
//}

func getDb() Database  {
	db,err := NewLevelDatabase(fPath)
	if err == nil {
	}
   return db
}


func putValue(b *testing.B, db SquashDatabase, key []byte, value []byte) {
	if db == nil {
		b.Fatal("the db is nil")
	}
	err := db.Put(key,value)
	if err != nil {
		b.Fatalf("put the key=%v , value=%v faile,error is %s", key, value, err)
	}
}

func get(b *testing.B, db SquashDatabase, key []byte) ([]byte,error) {
	if db == nil {
		b.Fatal("the db is nil")
	}
	if key == nil {
		b.Fatalf("the key=%s is nil",string(key))
	}
	val,err := db.Get(key)
	if err != nil {
		b.Fatalf("get the value of key=%s fail,error is %s",string(key),err)
		return nil,err
	}
	return val,nil
}

func deleteVal(b *testing.B, db SquashDatabase, key []byte) error {
	if db == nil {
		b.Fatal("the db is nil")
		return errors.New("the db is nil")
	}
	if key == nil {
		str := fmt.Sprintf("the key=%s is nil", string(key))
		b.Fatalf(str)
		return errors.New(str)
	}
	err := db.Delete(key)
	return err
}

func mdVal(b *testing.B, db SquashDatabase, key []byte, val []byte) error {
	if db == nil {
		b.Fatal("the db is nil")
		return errors.New("the db is nil")
	}
	if key == nil {
		str := fmt.Sprintf("the key=%s is nil", string(key))
		b.Fatalf(str)
		return errors.New(str)
	}
	err := db.Put(key,val)
	return err
}

func queryVal(b *testing.B, db SquashDatabase, key interface{}) error {
	if db == nil {
		b.Fatal("the db is nil")
		return errors.New("the db is nil")
	}
	if key == nil {
		b.Fatalf("can't encode nil key")
	}
	kVal,err := kope.Encode(key)
	if err != nil {
		return err
	}
	_,err = db.Get(kVal)
	return err
}