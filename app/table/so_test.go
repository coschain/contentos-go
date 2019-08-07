package table

import (
	"crypto/rand"
	"encoding/hex"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"os"
	"path/filepath"
	"testing"
)

func TestSoTable(t *testing.T) {
	a := assert.New(t)
	buf := make([]byte, 8)
	_, _ = rand.Reader.Read(buf)
	dbPath := filepath.Join(os.TempDir(), hex.EncodeToString(buf))
	db, err := storage.NewDatabase(dbPath)
	a.NoError(err)
	a.NoError(db.Start(nil))
	defer func() {
		db.Close()
		_ = os.RemoveAll(dbPath)
	}()

	t.Run("main", makeTestMain(db))
}

func makeTestMain(db *storage.DatabaseService) func(*testing.T) {
	return func(t *testing.T) {
		a := assert.New(t)

		// insert record "myName"
		mKey := prototype.NewAccountName("myName")
		wrap := NewSoDemoWrap(db, mKey)
		a.NotNil(wrap)
		a.NotPanics(func(){
			wrap.Create(func(tInfo *SoDemo) {
				tInfo.Owner = mKey
				tInfo.Title = "hello"
				tInfo.Content = "test the pb tool"
				tInfo.Idx = 1001
				tInfo.LikeCount = 100
				tInfo.Taglist = []string{"#NBA"}
				tInfo.ReplayCount = 100
				tInfo.PostTime = creTimeSecondPoint(20120401)
				tInfo.NickName = prototype.NewAccountName("jack")
				tInfo.RegistTime = prototype.NewTimePointSec(20120301)
			})
		})
		a.EqualValues("myName", wrap.GetOwner().Value)

		// insert record "myName1"
		key1 := prototype.NewAccountName("myName1")
		wrap1 := NewSoDemoWrap(db, key1)
		a.NotNil(wrap1)
		a.NotPanics(func(){
			wrap1.Create(func(tInfo *SoDemo) {
				tInfo.Owner = key1
				tInfo.Title = "hello1"
				tInfo.Content = "wrap1"
				tInfo.Idx = 1002
				tInfo.LikeCount = 200
				tInfo.Taglist = []string{"#Car"}
				tInfo.ReplayCount = 150
				tInfo.PostTime = creTimeSecondPoint(20120403)
				tInfo.NickName = prototype.NewAccountName("rose")
			})
		})
		// check members of record "myName1"
		a.EqualValues("myName1", wrap1.GetOwner().Value)
		a.EqualValues("wrap1", wrap1.GetContent())
		a.EqualValues(1002, wrap1.GetIdx())
		a.EqualValues(200, wrap1.GetLikeCount())

		// modify members
		a.EqualValues("test the pb tool", wrap.GetContent())
		a.EqualValues(1001, wrap.GetIdx())
		a.EqualValues(100, wrap.GetLikeCount())
		a.EqualValues("hello", wrap.GetTitle())
		a.NotPanics(func(){
			wrap.Modify(func(tInfo *SoDemo) {
				tInfo.Content = "hello world"
				tInfo.Idx = 1100
				tInfo.LikeCount = 10
				tInfo.Title = "test md title"
			})
		})
		a.EqualValues("hello world", wrap.GetContent())
		a.EqualValues(1100, wrap.GetIdx())
		a.EqualValues(10, wrap.GetLikeCount())
		a.EqualValues("test md title", wrap.GetTitle())

		// primary key is not allowed to be modified
		a.Panics(func(){
			wrap.Modify(func(tInfo *SoDemo) {
				tInfo.Owner = prototype.NewAccountName("test")
			})
		})

		// index member can't be set to nil
		a.Panics(func(){
			wrap.Modify(func(tInfo *SoDemo) {
				tInfo.PostTime = nil
			})
		})
		a.Panics(func(){
			wrap.Modify(func(tInfo *SoDemo) {
				tInfo.NickName = nil
			})
		})

		// if a member is not a key of any kind, it's ok to set it nil.
		a.NotPanics(func(){
			wrap.Modify(func(tInfo *SoDemo) {
				tInfo.RegistTime = nil
			})
		})
		a.NotPanics(func(){ wrap.SetContent("test md the content") })
		a.NotPanics(func(){ wrap.SetTaglist([]string{"#Football"}) })
		a.EqualValues("test md the content", wrap.GetContent())
		a.EqualValues([]string{"#Football"}, wrap.GetTaglist())

		var (
			err error
			result []string
		)

		// sort by post time ascending
		tSortWrap := NewDemoPostTimeWrap(db)
		err = tSortWrap.ForEachByOrder(creTimeSecondPoint(20120330), creTimeSecondPoint(20120415), nil ,nil,
			func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool {
				a.NotNil(mVal)
				a.NotNil(sVal)
				result = append(result, mVal.Value)
				return true
			})
		a.NoError(err)
		a.EqualValues([]string{"myName", "myName1"}, result)

		// sort by post time descending
		result = result[:0]
		err = tSortWrap.ForEachByRevOrder(creTimeSecondPoint(20120415), creTimeSecondPoint(20120330), nil ,nil,
			func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool {
				a.NotNil(mVal)
				a.NotNil(sVal)
				result = append(result, mVal.Value)
				return true
			})
		a.NoError(err)
		a.EqualValues([]string{"myName1", "myName"}, result)

		// sort by post time ascending, no start
		result = result[:0]
		err = tSortWrap.ForEachByOrder(nil, creTimeSecondPoint(20120415), nil ,nil,
			func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool {
				a.NotNil(mVal)
				a.NotNil(sVal)
				result = append(result, mVal.Value)
				return true
			})
		a.NoError(err)
		a.EqualValues([]string{"myName", "myName1"}, result)

		// sort by post time descending, no start
		result = result[:0]
		err = tSortWrap.ForEachByRevOrder(nil, creTimeSecondPoint(20120330), nil ,nil,
			func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool {
				a.NotNil(mVal)
				a.NotNil(sVal)
				result = append(result, mVal.Value)
				return true
			})
		a.NoError(err)
		a.EqualValues([]string{"myName1", "myName"}, result)

		// sort by post time ascending, no end
		result = result[:0]
		err = tSortWrap.ForEachByOrder(creTimeSecondPoint(20120330), nil, nil ,nil,
			func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool {
				a.NotNil(mVal)
				a.NotNil(sVal)
				result = append(result, mVal.Value)
				return true
			})
		a.NoError(err)
		a.EqualValues([]string{"myName", "myName1"}, result)

		// sort by post time descending, no end
		result = result[:0]
		err = tSortWrap.ForEachByRevOrder(creTimeSecondPoint(20120415), nil, nil ,nil,
			func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool {
				a.NotNil(mVal)
				a.NotNil(sVal)
				result = append(result, mVal.Value)
				return true
			})
		a.NoError(err)
		a.EqualValues([]string{"myName1", "myName"}, result)

		// sort by post time ascending, no start nor end
		result = result[:0]
		err = tSortWrap.ForEachByOrder(nil, nil, nil ,nil,
			func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool {
				a.NotNil(mVal)
				a.NotNil(sVal)
				result = append(result, mVal.Value)
				return true
			})
		a.NoError(err)
		a.EqualValues([]string{"myName", "myName1"}, result)

		// sort by post time descending, no start nor end
		result = result[:0]
		err = tSortWrap.ForEachByRevOrder(nil, nil, nil ,nil,
			func(mVal *prototype.AccountName, sVal *prototype.TimePointSec, idx uint32) bool {
				a.NotNil(mVal)
				a.NotNil(sVal)
				result = append(result, mVal.Value)
				return true
			})
		a.NoError(err)
		a.EqualValues([]string{"myName1", "myName"}, result)

		// query by unique key
		idx := int64(1100)
		uniWrap := NewUniDemoIdxWrap(db)
		dWrap := uniWrap.UniQueryIdx(&idx)
		a.NotNil(dWrap)
		a.EqualValues("myName", dWrap.GetOwner().Value)
		idx = 1002
		a.EqualValues("myName1", uniWrap.UniQueryIdx(&idx).GetOwner().Value)

		// remove
		a.NotPanics(func(){ wrap.MustExist() })
		a.NotPanics(func(){ wrap.RemoveDemo() })
		a.NotPanics(func(){ wrap.MustNotExist() })

		a.NotPanics(func(){ wrap1.MustExist() })
		a.NotPanics(func(){ wrap1.RemoveDemo() })
		a.NotPanics(func(){ wrap1.MustNotExist() })
	}
}

func creTimeSecondPoint(t uint32) *prototype.TimePointSec {
	val := prototype.TimePointSec{UtcSeconds: t}
	return &val
}
