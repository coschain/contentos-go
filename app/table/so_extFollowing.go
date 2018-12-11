package table

import (
	"errors"
	fmt "fmt"
	"reflect"
	"strings"

	"github.com/coschain/contentos-go/common/encoding/kope"
	"github.com/coschain/contentos-go/iservices"
	prototype "github.com/coschain/contentos-go/prototype"
	proto "github.com/golang/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	ExtFollowingTable                      = []byte("ExtFollowingTable")
	ExtFollowingFollowingCreatedOrderTable = []byte("ExtFollowingFollowingCreatedOrderTable")
	ExtFollowingFollowingInfoUniTable      = []byte("ExtFollowingFollowingInfoUniTable")
)

////////////// SECTION Wrap Define ///////////////
type SoExtFollowingWrap struct {
	dba     iservices.IDatabaseService
	mainKey *prototype.FollowingRelation
}

func NewSoExtFollowingWrap(dba iservices.IDatabaseService, key *prototype.FollowingRelation) *SoExtFollowingWrap {
	if dba == nil || key == nil {
		return nil
	}
	result := &SoExtFollowingWrap{dba, key}
	return result
}

func (s *SoExtFollowingWrap) CheckExist() bool {
	if s.dba == nil {
		return false
	}
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}

	res, err := s.dba.Has(keyBuf)
	if err != nil {
		return false
	}

	return res
}

func (s *SoExtFollowingWrap) Create(f func(tInfo *SoExtFollowing)) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if s.mainKey == nil {
		return errors.New("the main key is nil")
	}
	val := &SoExtFollowing{}
	f(val)
	if val.FollowingInfo == nil {
		val.FollowingInfo = s.mainKey
	}
	if s.CheckExist() {
		return errors.New("the main key is already exist")
	}
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return err

	}
	err = s.saveAllMemKeys(val, true)
	if err != nil {
		return err
	}

	// update sort list keys
	if err = s.insertAllSortKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.dba.Delete(keyBuf)
		s.delAllMemKeys(false, val)
		return err
	}

	//update unique list
	if sucNames, err := s.insertAllUniKeys(val); err != nil {
		s.delAllSortKeys(false, val)
		s.delUniKeysWithNames(sucNames, val)
		s.dba.Delete(keyBuf)
		s.delAllMemKeys(false, val)
		return err
	}

	return nil
}

func (s *SoExtFollowingWrap) encodeMemKey(fName string) ([]byte, error) {
	if len(fName) < 1 || s.mainKey == nil {
		return nil, errors.New("field name or main key is empty")
	}
	pre := "ExtFollowing" + fName + "cell"
	kList := []interface{}{pre, s.mainKey}
	key, err := kope.EncodeSlice(kList)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func (so *SoExtFollowingWrap) saveAllMemKeys(tInfo *SoExtFollowing, br bool) error {
	if so.dba == nil {
		return errors.New("save member Field fail , the db is nil")
	}

	if tInfo == nil {
		return errors.New("save member Field fail , the data is nil ")
	}
	var err error = nil
	errDes := ""
	if err = so.saveMemKeyFollowingCreatedOrder(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "FollowingCreatedOrder", err)
		}
	}
	if err = so.saveMemKeyFollowingInfo(tInfo); err != nil {
		if br {
			return err
		} else {
			errDes += fmt.Sprintf("save the Field %s fail,error is %s;\n", "FollowingInfo", err)
		}
	}

	if len(errDes) > 0 {
		return errors.New(errDes)
	}
	return err
}

func (so *SoExtFollowingWrap) delAllMemKeys(br bool, tInfo *SoExtFollowing) error {
	if so.dba == nil {
		return errors.New("the db is nil")
	}
	t := reflect.TypeOf(*tInfo)
	errDesc := ""
	for k := 0; k < t.NumField(); k++ {
		name := t.Field(k).Name
		if len(name) > 0 && !strings.HasPrefix(name, "XXX_") {
			err := so.delMemKey(name)
			if err != nil {
				if br {
					return err
				}
				errDesc += fmt.Sprintf("delete the Field %s fail,error is %s;\n", name, err)
			}
		}
	}
	if len(errDesc) > 0 {
		return errors.New(errDesc)
	}
	return nil
}

func (so *SoExtFollowingWrap) delMemKey(fName string) error {
	if so.dba == nil {
		return errors.New("the db is nil")
	}
	if len(fName) <= 0 {
		return errors.New("the field name is empty ")
	}
	key, err := so.encodeMemKey(fName)
	if err != nil {
		return err
	}
	err = so.dba.Delete(key)
	return err
}

////////////// SECTION LKeys delete/insert ///////////////

func (s *SoExtFollowingWrap) delSortKeyFollowingCreatedOrder(sa *SoExtFollowing) bool {
	if s.dba == nil || s.mainKey == nil {
		return false
	}
	val := SoListExtFollowingByFollowingCreatedOrder{}
	if sa == nil {
		key, err := s.encodeMemKey("FollowingCreatedOrder")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemExtFollowingByFollowingCreatedOrder{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		val.FollowingCreatedOrder = ori.FollowingCreatedOrder
		val.FollowingInfo = s.mainKey

	} else {
		val.FollowingCreatedOrder = sa.FollowingCreatedOrder
		val.FollowingInfo = sa.FollowingInfo
	}

	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Delete(subBuf)
	return ordErr == nil
}

func (s *SoExtFollowingWrap) insertSortKeyFollowingCreatedOrder(sa *SoExtFollowing) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	val := SoListExtFollowingByFollowingCreatedOrder{}
	val.FollowingInfo = sa.FollowingInfo
	val.FollowingCreatedOrder = sa.FollowingCreatedOrder
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
	subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
	ordErr := s.dba.Put(subBuf, buf)
	return ordErr == nil
}

func (s *SoExtFollowingWrap) delAllSortKeys(br bool, val *SoExtFollowing) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delSortKeyFollowingCreatedOrder(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtFollowingWrap) insertAllSortKeys(val *SoExtFollowing) error {
	if s.dba == nil {
		return errors.New("insert sort Field fail,the db is nil ")
	}
	if val == nil {
		return errors.New("insert sort Field fail,get the SoExtFollowing fail ")
	}
	if !s.insertSortKeyFollowingCreatedOrder(val) {
		return errors.New("insert sort Field FollowingCreatedOrder fail while insert table ")
	}

	return nil
}

////////////// SECTION LKeys delete/insert //////////////

func (s *SoExtFollowingWrap) RemoveExtFollowing() bool {
	if s.dba == nil {
		return false
	}
	val := &SoExtFollowing{}
	//delete sort list key
	if res := s.delAllSortKeys(true, nil); !res {
		return false
	}

	//delete unique list
	if res := s.delAllUniKeys(true, nil); !res {
		return false
	}

	err := s.delAllMemKeys(true, val)
	return err == nil
}

////////////// SECTION Members Get/Modify ///////////////
func (s *SoExtFollowingWrap) saveMemKeyFollowingCreatedOrder(tInfo *SoExtFollowing) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtFollowingByFollowingCreatedOrder{}
	val.FollowingCreatedOrder = tInfo.FollowingCreatedOrder
	key, err := s.encodeMemKey("FollowingCreatedOrder")
	if err != nil {
		return err
	}
	buf, err := proto.Marshal(&val)
	if err != nil {
		return err
	}
	err = s.dba.Put(key, buf)
	return err
}

func (s *SoExtFollowingWrap) GetFollowingCreatedOrder() *prototype.FollowingCreatedOrder {
	res := true
	msg := &SoMemExtFollowingByFollowingCreatedOrder{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("FollowingCreatedOrder")
		if err != nil {
			res = false
		} else {
			buf, err := s.dba.Get(key)
			if err != nil {
				res = false
			}
			err = proto.Unmarshal(buf, msg)
			if err != nil {
				res = false
			} else {
				return msg.FollowingCreatedOrder
			}
		}
	}
	if !res {
		return nil

	}
	return msg.FollowingCreatedOrder
}

func (s *SoExtFollowingWrap) MdFollowingCreatedOrder(p *prototype.FollowingCreatedOrder) bool {
	if s.dba == nil {
		return false
	}
	key, err := s.encodeMemKey("FollowingCreatedOrder")
	if err != nil {
		return false
	}
	buf, err := s.dba.Get(key)
	if err != nil {
		return false
	}
	ori := &SoMemExtFollowingByFollowingCreatedOrder{}
	err = proto.Unmarshal(buf, ori)
	sa := &SoExtFollowing{}
	sa.FollowingInfo = s.mainKey

	sa.FollowingCreatedOrder = ori.FollowingCreatedOrder

	if !s.delSortKeyFollowingCreatedOrder(sa) {
		return false
	}
	ori.FollowingCreatedOrder = p
	val, err := proto.Marshal(ori)
	if err != nil {
		return false
	}
	err = s.dba.Put(key, val)
	if err != nil {
		return false
	}
	sa.FollowingCreatedOrder = p

	if !s.insertSortKeyFollowingCreatedOrder(sa) {
		return false
	}

	return true
}

func (s *SoExtFollowingWrap) saveMemKeyFollowingInfo(tInfo *SoExtFollowing) error {
	if s.dba == nil {
		return errors.New("the db is nil")
	}
	if tInfo == nil {
		return errors.New("the data is nil")
	}
	val := SoMemExtFollowingByFollowingInfo{}
	val.FollowingInfo = tInfo.FollowingInfo
	key, err := s.encodeMemKey("FollowingInfo")
	if err != nil {
		return err
	}
	buf, err := proto.Marshal(&val)
	if err != nil {
		return err
	}
	err = s.dba.Put(key, buf)
	return err
}

func (s *SoExtFollowingWrap) GetFollowingInfo() *prototype.FollowingRelation {
	res := true
	msg := &SoMemExtFollowingByFollowingInfo{}
	if s.dba == nil {
		res = false
	} else {
		key, err := s.encodeMemKey("FollowingInfo")
		if err != nil {
			res = false
		} else {
			buf, err := s.dba.Get(key)
			if err != nil {
				res = false
			}
			err = proto.Unmarshal(buf, msg)
			if err != nil {
				res = false
			} else {
				return msg.FollowingInfo
			}
		}
	}
	if !res {
		return nil

	}
	return msg.FollowingInfo
}

////////////// SECTION List Keys ///////////////
type SExtFollowingFollowingCreatedOrderWrap struct {
	Dba iservices.IDatabaseService
}

func NewExtFollowingFollowingCreatedOrderWrap(db iservices.IDatabaseService) *SExtFollowingFollowingCreatedOrderWrap {
	if db == nil {
		return nil
	}
	wrap := SExtFollowingFollowingCreatedOrderWrap{Dba: db}
	return &wrap
}

func (s *SExtFollowingFollowingCreatedOrderWrap) DelIterater(iterator iservices.IDatabaseIterator) {
	if iterator == nil || !iterator.Valid() {
		return
	}
	s.Dba.DeleteIterator(iterator)
}

func (s *SExtFollowingFollowingCreatedOrderWrap) GetMainVal(iterator iservices.IDatabaseIterator) *prototype.FollowingRelation {
	if iterator == nil || !iterator.Valid() {
		return nil
	}
	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoListExtFollowingByFollowingCreatedOrder{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
	return res.FollowingInfo

}

func (s *SExtFollowingFollowingCreatedOrderWrap) GetSubVal(iterator iservices.IDatabaseIterator) *prototype.FollowingCreatedOrder {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}
	res := &SoListExtFollowingByFollowingCreatedOrder{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
	return res.FollowingCreatedOrder

}

func (m *SoListExtFollowingByFollowingCreatedOrder) OpeEncode() ([]byte, error) {
	pre := ExtFollowingFollowingCreatedOrderTable
	sub := m.FollowingCreatedOrder
	if sub == nil {
		return nil, errors.New("the pro FollowingCreatedOrder is nil")
	}
	sub1 := m.FollowingInfo
	if sub1 == nil {
		return nil, errors.New("the mainkey FollowingInfo is nil")
	}
	kList := []interface{}{pre, sub, sub1}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

//Query sort by order
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *SExtFollowingFollowingCreatedOrderWrap) QueryListByOrder(start *prototype.FollowingCreatedOrder, end *prototype.FollowingCreatedOrder) iservices.IDatabaseIterator {
	if s.Dba == nil {
		return nil
	}
	pre := ExtFollowingFollowingCreatedOrderTable
	skeyList := []interface{}{pre}
	if start != nil {
		skeyList = append(skeyList, start)
	}
	sBuf, cErr := kope.EncodeSlice(skeyList)
	if cErr != nil {
		return nil
	}
	eKeyList := []interface{}{pre}
	if end != nil {
		eKeyList = append(eKeyList, end)
	} else {
		eKeyList = append(eKeyList, kope.MaximumKey)
	}
	eBuf, cErr := kope.EncodeSlice(eKeyList)
	if cErr != nil {
		return nil
	}
	return s.Dba.NewIterator(sBuf, eBuf)
}

/////////////// SECTION Private function ////////////////

func (s *SoExtFollowingWrap) update(sa *SoExtFollowing) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	buf, err := proto.Marshal(sa)
	if err != nil {
		return false
	}

	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}

	return s.dba.Put(keyBuf, buf) == nil
}

func (s *SoExtFollowingWrap) getExtFollowing() *SoExtFollowing {
	if s.dba == nil {
		return nil
	}
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return nil
	}
	resBuf, err := s.dba.Get(keyBuf)

	if err != nil {
		return nil
	}

	res := &SoExtFollowing{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}

func (s *SoExtFollowingWrap) encodeMainKey() ([]byte, error) {
	pre := "ExtFollowing" + "FollowingInfo" + "cell"
	sub := s.mainKey
	if sub == nil {
		return nil, errors.New("the mainKey is nil")
	}
	kList := []interface{}{pre, sub}
	kBuf, cErr := kope.EncodeSlice(kList)
	return kBuf, cErr
}

////////////// Unique Query delete/insert/query ///////////////

func (s *SoExtFollowingWrap) delAllUniKeys(br bool, val *SoExtFollowing) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if !s.delUniKeyFollowingInfo(val) {
		if br {
			return false
		} else {
			res = false
		}
	}

	return res
}

func (s *SoExtFollowingWrap) delUniKeysWithNames(names map[string]string, val *SoExtFollowing) bool {
	if s.dba == nil {
		return false
	}
	res := true
	if len(names["FollowingInfo"]) > 0 {
		if !s.delUniKeyFollowingInfo(val) {
			res = false
		}
	}

	return res
}

func (s *SoExtFollowingWrap) insertAllUniKeys(val *SoExtFollowing) (map[string]string, error) {
	if s.dba == nil {
		return nil, errors.New("insert uniuqe Field fail,the db is nil ")
	}
	if val == nil {
		return nil, errors.New("insert uniuqe Field fail,get the SoExtFollowing fail ")
	}
	sucFields := map[string]string{}
	if !s.insertUniKeyFollowingInfo(val) {
		return sucFields, errors.New("insert unique Field FollowingInfo fail while insert table ")
	}
	sucFields["FollowingInfo"] = "FollowingInfo"

	return sucFields, nil
}

func (s *SoExtFollowingWrap) delUniKeyFollowingInfo(sa *SoExtFollowing) bool {
	if s.dba == nil {
		return false
	}
	pre := ExtFollowingFollowingInfoUniTable
	kList := []interface{}{pre}
	if sa != nil {

		if sa.FollowingInfo == nil {
			return false
		}

		sub := sa.FollowingInfo
		kList = append(kList, sub)
	} else {
		key, err := s.encodeMemKey("FollowingInfo")
		if err != nil {
			return false
		}
		buf, err := s.dba.Get(key)
		if err != nil {
			return false
		}
		ori := &SoMemExtFollowingByFollowingInfo{}
		err = proto.Unmarshal(buf, ori)
		if err != nil {
			return false
		}
		sub := ori.FollowingInfo
		kList = append(kList, sub)

	}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}

func (s *SoExtFollowingWrap) insertUniKeyFollowingInfo(sa *SoExtFollowing) bool {
	if s.dba == nil || sa == nil {
		return false
	}
	uniWrap := UniExtFollowingFollowingInfoWrap{}
	uniWrap.Dba = s.dba

	res := uniWrap.UniQueryFollowingInfo(sa.FollowingInfo)
	if res != nil {
		//the unique key is already exist
		return false
	}
	val := SoUniqueExtFollowingByFollowingInfo{}
	val.FollowingInfo = sa.FollowingInfo

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	pre := ExtFollowingFollowingInfoUniTable
	sub := sa.FollowingInfo
	kList := []interface{}{pre, sub}
	kBuf, err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type UniExtFollowingFollowingInfoWrap struct {
	Dba iservices.IDatabaseService
}

func NewUniExtFollowingFollowingInfoWrap(db iservices.IDatabaseService) *UniExtFollowingFollowingInfoWrap {
	if db == nil {
		return nil
	}
	wrap := UniExtFollowingFollowingInfoWrap{Dba: db}
	return &wrap
}

func (s *UniExtFollowingFollowingInfoWrap) UniQueryFollowingInfo(start *prototype.FollowingRelation) *SoExtFollowingWrap {
	if start == nil || s.Dba == nil {
		return nil
	}
	pre := ExtFollowingFollowingInfoUniTable
	kList := []interface{}{pre, start}
	bufStartkey, err := kope.EncodeSlice(kList)
	val, err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUniqueExtFollowingByFollowingInfo{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			wrap := NewSoExtFollowingWrap(s.Dba, res.FollowingInfo)

			return wrap
		}
	}
	return nil
}
