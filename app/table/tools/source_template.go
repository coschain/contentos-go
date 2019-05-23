package main

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"hash/crc32"
	"log"
	"os/exec"
	"reflect"
	"regexp"
	"strings"
	"text/template"
	"unicode"
)

var prefixMap = map[uint32]string{}

type FieldType int
const (
	FieldTypeRow  FieldType = iota //common member field in the table
	FieldTypeSort                  //a field which is supported sorting
	FieldTypeUni                   //a field which is supported unique query
)

//the base field of a table
type Field struct {
	PName string   //the name of the field
	PType string   //the type of the field
	Prefix uint32  //the prefix for a key in the database
}

//the member field of a table
type SortField struct {
    Field
	SType int //1:support order 2:support reverse order 3:support order and reverse order
}


type Params struct {
	ClsName     string
	MainKeyType string
	MainKeyName string

	LKeys          []string
	MemberKeyMap   map[string]Field
	LKeyWithType   map[string]string
	UniqueFieldMap map[string]Field
	SortList       []SortField
	SListCount     int
}

func CreateGoFile(tInfo TableInfo) (bool, error) {
	var err error = nil
	if tInfo.Name == "" {
		err = errors.New("table name is empty")
		return false, err
	} else if len(tInfo.PList) < 1 {
		err = errors.New("table data is empty")
		return false, err
	}

	tmpl := `

package table


////////////// SECTION Prefix Mark ///////////////
var (
    {{range $k, $v := .SortList -}}
    {{$.ClsName}}{{$v.PName}}Table uint32 = {{$v.Prefix}}
    {{end -}}
    {{range $k, $v := .UniqueFieldMap -}}
	{{$.ClsName}}{{$k}}UniTable uint32 = {{$v.Prefix}}
    {{end -}}
    {{range $k, $v := .MemberKeyMap -}}
    {{if eq $k $.MainKeyName}}
    {{$.ClsName}}{{$k}}Row uint32 = {{$v.Prefix}}
    {{end}}
    {{end -}}
)

////////////// SECTION Wrap Define ///////////////
type So{{.ClsName}}Wrap struct {
	dba 		iservices.IDatabaseRW
	mainKey 	*{{formatStr .MainKeyType}}
    mKeyFlag    int //the flag of the main key exist state in db, -1:has not judged; 0:not exist; 1:already exist
	mKeyBuf     []byte //the buffer after the main key is encoded with prefix
	mBuf        []byte //the value after the main key is encoded
    mdFuncMap  map[string]interface{}
}

func NewSo{{.ClsName}}Wrap(dba iservices.IDatabaseRW, key *{{formatStr .MainKeyType}}) *So{{.ClsName}}Wrap{
	if dba == nil || key == nil {
       return nil
    }
    result := &So{{.ClsName}}Wrap{dba,key,-1,nil,nil, nil}
	return result
}

func (s *So{{.ClsName}}Wrap) CheckExist() bool {
    if s.dba ==  nil {
       return false
    }
    if s.mKeyFlag != -1 {
		//if you have already obtained the existence status of the primary key, use it directly
		if s.mKeyFlag == 0 {
			return false
		}
		return true
	}
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return false
	}

	res, err := s.dba.Has(keyBuf)
	if err != nil {
		return false
	}
    if res == false {
    	s.mKeyFlag = 0
	}else {
		s.mKeyFlag = 1
	}
	return res
}

func (s *So{{.ClsName}}Wrap) Create(f func(tInfo *So{{.ClsName}})) error {
    if s.dba == nil {
       return errors.New("the db is nil")
    }
    if s.mainKey == nil {
          return errors.New("the main key is nil")
    }
    val := &So{{.ClsName}}{}
    f(val)
    {{$baseType := (DetectBaseType $.MainKeyType) -}}
    {{- if not $baseType -}}
    if val.{{$.MainKeyName}} == nil {
       val.{{$.MainKeyName}} = s.mainKey
    }
    {{ end -}}
    if s.CheckExist() {
       return errors.New("the main key is already exist")
    }
	keyBuf, err := s.encodeMainKey()
	if err != nil {
       return err

	}
        
    buf,err := proto.Marshal(val)
	if err != nil {
		return err
	}
	err = s.dba.Put(keyBuf, buf)
	if err != nil {
		return err
	}


    {{if ge  $.SListCount 0 -}}
	// update srt list keys
	if err = s.insertAllSortKeys(val); err != nil {
       s.delAllSortKeys(false,val)
       s.dba.Delete(keyBuf)
       return err
    }
    {{end}}
    {{if ge (getMapCount .UniqueFieldMap) 0 -}}
    //update unique list
    if sucNames,err := s.insertAllUniKeys(val); err != nil {
        s.delAllSortKeys(false,val)
        s.delUniKeysWithNames(sucNames,val)
        s.dba.Delete(keyBuf)
        return err
    }
    {{end}}
    s.mKeyFlag = 1
	return nil
}

func (s *So{{.ClsName}}Wrap) getMainKeyBuf() ([]byte, error) {
	if s.mainKey == nil {
		return nil,errors.New("the main key is nil")
	}
	if s.mBuf == nil {
		var err error = nil
		s.mBuf,err = kope.Encode(s.mainKey)
		if err != nil {
			return nil,err
		}
	}
	return s.mBuf,nil
}

func (s *So{{.ClsName}}Wrap) Md(f func(tInfo *So{{.ClsName}})) error {
    if !s.CheckExist() {
		return errors.New("the So{{.ClsName}} table does not exist. Please create a table first")
	}
    oriTable := s.get{{.ClsName}}()
	if oriTable == nil {
		return errors.New("fail to get origin table So{{.ClsName}}")
	}
	curTable := *oriTable
	f(&curTable)

    //the main key is not support modify
    if !reflect.DeepEqual(curTable.{{$.MainKeyName}}, oriTable.{{$.MainKeyName}}) {
       curTable.{{$.MainKeyName}} = oriTable.{{$.MainKeyName}}
    }

    fieldSli,err := s.getModifiedFields(oriTable, &curTable)
    if err != nil {
		return err
	}
 
    if fieldSli == nil || len(fieldSli) < 1 {
		return nil
	}

    //check unique 
    err = s.handleFieldMd(FieldMdHandleTypeCheck, &curTable, fieldSli)
    if err != nil {
       return err
    }
      
    //delete sort and unique key
    err = s.handleFieldMd(FieldMdHandleTypeDel, oriTable, fieldSli)
    if err != nil {
       return err
    }
    
    //update table
    err = s.update{{.ClsName}}(&curTable)
    if err != nil {
       return err
    }

    //insert sort and unique key 
    err = s.handleFieldMd(FieldMdHandleTypeInsert, &curTable, fieldSli)
    if err != nil {
       return err
    }
    
    return nil

}


//Get all the modified fields in the table
func (s *So{{$.ClsName}}Wrap) getModifiedFields (oriTable *So{{$.ClsName}}, curTable *So{{$.ClsName}}) ([]string, error) {
     if oriTable == nil  {
       return nil,errors.New("table info is nil, can't get modified fields")
	 }
     var list []string
     {{range $k1, $v1 := .MemberKeyMap}}
     {{if ne $k1 $.MainKeyName}}
     if !reflect.DeepEqual(oriTable.{{$k1}}, curTable.{{$k1}}) {
		list = append(list, "{{$k1}}")
	 }
     {{end}}
     {{end}}
     return list,nil
}

func (s *So{{$.ClsName}}Wrap) handleFieldMd (t FieldMdHandleType, so *So{{.ClsName}}, fSli []string) error {
     if so == nil {
        return errors.New("fail to modify empty table")
     }
     
     //there is no field need to modify
     if fSli == nil || len(fSli) < 1 {
        return nil
     }


     errStr := ""
     for _,fName := range fSli {
        {{range $k1, $v1 := .MemberKeyMap}}
        {{if ne $k1 $.MainKeyName}}
         if fName == "{{$k1}}" {
             res := true
            if t == FieldMdHandleTypeCheck {
               res = s.mdField{{$k1}}(so.{{$k1}}, true, false, false, so)
               errStr = fmt.Sprintf("fail to modify exist value of %v", fName)
            } else if t == FieldMdHandleTypeDel {
               res = s.mdField{{$k1}}(so.{{$k1}}, false, true, false, so)
               errStr = fmt.Sprintf("fail to delete  sort or unique field  %v", fName)
            } else if t == FieldMdHandleTypeInsert {
               res = s.mdField{{$k1}}(so.{{$k1}}, false, false, true, so)
               errStr = fmt.Sprintf("fail to insert  sort or unique field  %v", fName)
            } 
            if !res {
               return errors.New(errStr)
            }
         } 
         {{end}}
         {{end}}
     }
     
     return nil
}



////////////// SECTION LKeys delete/insert ///////////////
{{range $k1, $v1 := .SortList}}
func (s *So{{$.ClsName}}Wrap) delSortKey{{$v1.PName}}(sa *So{{$.ClsName}}) bool {
    if s.dba == nil || s.mainKey == nil{
       return false
    }
	val := SoList{{$.ClsName}}By{{$v1.PName}}{}
    if sa == nil {
       val.{{$v1.PName}} = s.Get{{$v1.PName}}()
       {{if ne $.MainKeyName $v1.PName -}}
       {{ $baseType := (DetectBaseType $.MainKeyType) -}}
       {{- if $baseType -}} 
       val.{{UperFirstChar $.MainKeyName}} = *s.mainKey
       {{- end -}}
       {{if not $baseType -}} 
   	   val.{{UperFirstChar $.MainKeyName}} = s.mainKey
       {{end }}
       {{end -}}
    }else {
       val.{{$v1.PName}} = sa.{{$v1.PName}}
       {{if ne $.MainKeyName $v1.PName -}}
       val.{{UperFirstChar $.MainKeyName}} = sa.{{UperFirstChar $.MainKeyName}}
       {{end -}}
    }
	{{$baseType := (DetectBaseType $v1.PType) -}}
    {{- if not $baseType -}}
    if val.{{$v1.PName}} == nil {
       return true
    }
    {{ end -}}
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordErr :=  s.dba.Delete(subBuf)
    return ordErr == nil
}


func (s *So{{$.ClsName}}Wrap) insertSortKey{{$v1.PName}}(sa *So{{$.ClsName}}) bool {
    if s.dba == nil || sa == nil {
       return false
    }
    {{$baseType := (DetectBaseType $v1.PType) -}}
    {{- if not $baseType -}}
    if sa.{{$v1.PName}} == nil {
       return true
    }
    {{ end -}}
	val := SoList{{$.ClsName}}By{{$v1.PName}}{}
    {{if ne $.MainKeyName $v1.PName -}}
   	val.{{UperFirstChar $.MainKeyName}} = sa.{{UperFirstChar $.MainKeyName}}
    {{end -}}
	val.{{UperFirstChar $v1.PName}} = sa.{{UperFirstChar $v1.PName}}
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordErr :=  s.dba.Put(subBuf, buf) 
    return ordErr == nil
}

{{end}}

{{if ge .SListCount 0}}
func (s *So{{$.ClsName}}Wrap) delAllSortKeys(br bool, val *So{{.ClsName}}) bool {
    if s.dba == nil {
       return false
    }
    res := true
    {{range $k, $v := .LKeys -}}
    {{if ne $v $.MainKeyName}}
    if !s.delSortKey{{$v}}(val) {
        if br {
           return false
        }else {
           res = false
        }
    }
    {{end}}
	{{end}}
    return res
}

func (s *So{{$.ClsName}}Wrap)insertAllSortKeys(val *So{{$.ClsName}}) error {
    if s.dba == nil {
       return errors.New("insert sort Field fail,the db is nil ")
    }
	if val == nil {
		return errors.New("insert sort Field fail,get the So{{.ClsName}} fail ")
	}
    {{range $k, $v := .LKeys -}}
    {{if ne $v $.MainKeyName}}
    if !s.insertSortKey{{$v}}(val) {
       return errors.New("insert sort Field {{$v}} fail while insert table ")
	}
    {{end}}
	{{end}}    
    return nil
}
{{end}}

////////////// SECTION LKeys delete/insert //////////////

func (s *So{{.ClsName}}Wrap) Remove{{.ClsName}}() bool {
    if s.dba == nil {
       return false
    }
    {{if ge  $.SListCount 0 -}}
    //delete sort list key
    if res := s.delAllSortKeys(true, nil); !res {
       return false
    }
    {{end}}
    {{if ge (getMapCount .UniqueFieldMap) 0 -}}
    //delete unique list
    if res := s.delAllUniKeys(true,nil);  !res {
       return false
    }
    {{end}}

    //delete table 
    key,err := s.encodeMainKey()
    if err != nil {
       return false
    }
    err = s.dba.Delete(key)
    if err == nil {
		s.mKeyBuf = nil
		s.mKeyFlag = -1
		return true
	}else{
		return false
	}
}

////////////// SECTION Members Get/Modify ///////////////

{{range $k1, $v1 := .MemberKeyMap -}}

func (s *So{{$.ClsName}}Wrap) Get{{$k1}}() {{formatRTypeStr $v1.PType}} {
   res := true
   msg := &So{{$.ClsName}}{}
   if s.dba == nil { 
      res = false
   }else {
      key,err := s.encodeMainKey()
      if err != nil {
         res = false
      }else { 
          buf,err := s.dba.Get(key)
          if err != nil {
             res = false
          }
          err = proto.Unmarshal(buf, msg)
          if err != nil {
             res = false
          }else {
             return msg.{{$k1}}
          }
      }
   }
   if !res {
      {{$baseType := (DetectBaseType $v1.PType) -}}
      {{- if $baseType -}} 
      var tmpValue {{formatRTypeStr $v1.PType}} 
      return tmpValue
      {{- end -}}
      {{if not $baseType -}} 
      return nil
      {{end}}
   }
   return msg.{{$k1}}
}

{{if ne $k1 $.MainKeyName}}
func (s *So{{$.ClsName}}Wrap) mdField{{$k1}}(p {{formatRTypeStr $v1.PType}}, isCheck bool, isDel bool, isInsert bool, 
 so *So{{$.ClsName}}) bool {
	if s.dba == nil {
		return false
	}
	
	if isCheck {
		res := s.check{{$k1}}IsMetMdCondition(p)
		if !res {
			return false
		}
	}
	
	if isDel {
		res := s.delField{{$k1}}(so)
		if !res {
			return false
		}
	}
	
	if isInsert {
		res := s.insertField{{$k1}}(so)
		if !res {
			return false
		}
	}
	return true
}

func (s *So{{$.ClsName}}Wrap) delField{{$k1}}(so *So{{$.ClsName}}) bool {
   if s.dba == nil {
      return false
   }
   {{- range $k2, $v2 := $.UniqueFieldMap -}}
   {{- if eq $k2 $k1 }}
   if !s.delUniKey{{$k2}}(so) {
		return false
   }
   {{end}}
   {{end}}

   {{range $k3, $v3 := $.LKeys -}}
   {{if eq $v3 $k1 }}
   if !s.delSortKey{{$k1}}(so) {
      return false
   }
   {{- end -}}
   {{end}}
   
   return true
}

func (s *So{{$.ClsName}}Wrap) insertField{{$k1}}(so *So{{$.ClsName}}) bool {
	if s.dba == nil {
		return false
	}
   {{range $k2, $v2 := $.LKeys -}}
   {{if eq $v2 $k1}}
   if !s.insertSortKey{{$k1}}(so) {
		return false
   }
   {{end}}
   {{end}}

   {{- range $k3, $v3 := $.UniqueFieldMap}}
		{{if eq $k3 $k1 }}
   if !s.insertUniKey{{$k3}}(so) {
		return false
   }
   {{- end -}}
   {{end}}
   return true
}

func (s *So{{$.ClsName}}Wrap) check{{$k1}}IsMetMdCondition(p {{formatRTypeStr $v1.PType}}) bool {
	if s.dba == nil {
		return false
	}
    {{- range $k2, $v2 := $.UniqueFieldMap -}}
      {{- if eq $k2 $k1 }}
    //judge the unique value if is exist
    uniWrap  := Uni{{$.ClsName}}{{$k2}}Wrap{}
    uniWrap.Dba = s.dba
   {{ $baseType := (DetectBaseType $v2.PType) -}}
   {{- if $baseType -}} 
   	res := uniWrap.UniQuery{{$k1}}(&p)
   {{- end -}}
   {{if not $baseType -}} 
   	res := uniWrap.UniQuery{{$k1}}(p)
   {{end }}
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}
   {{end}}
   {{end}}
	return true
}

{{end}}
{{end}}

{{range $k, $v := .SortList}}
////////////// SECTION List Keys ///////////////
type S{{$.ClsName}}{{$v.PName}}Wrap struct {
	Dba iservices.IDatabaseRW
}

func New{{$.ClsName}}{{$v.PName}}Wrap(db iservices.IDatabaseRW) *S{{$.ClsName}}{{$v.PName}}Wrap {
     if db == nil {
        return nil
     }
     wrap := S{{$.ClsName}}{{$v.PName}}Wrap{Dba:db}
     return &wrap
}

func (s *S{{$.ClsName}}{{$v.PName}}Wrap) GetMainVal(val []byte) *{{formatStr $.MainKeyType}} {
	res := &SoList{{$.ClsName}}By{{$v.PName}}{}
	err := proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    {{$baseType := (DetectBaseType $.MainKeyType) -}}
   {{if $baseType}} 
     return &res.{{$.MainKeyName}}
   {{end -}}
   {{if not $baseType -}} 
   return res.{{$.MainKeyName}}
   {{end}}
}

func (s *S{{$.ClsName}}{{$v.PName}}Wrap) GetSubVal(val []byte) *{{formatSliceType $v.PType}} {
	res := &SoList{{$.ClsName}}By{{$v.PName}}{}
	err := proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
    {{$baseType := (DetectBaseType $v.PType) -}}
   {{if $baseType -}} 
     return &res.{{ $v.PName}}
   {{end -}}
   {{if not $baseType -}} 
   return res.{{$v.PName}}
   {{end }}
}

func (m *SoList{{$.ClsName}}By{{$v.PName}}) OpeEncode() ([]byte,error) {
    pre := {{$.ClsName}}{{$v.PName}}Table
    sub := m.{{$v.PName}}
    {{$baseType := (DetectBaseType $v.PType) -}}
    {{- if not $baseType -}} 
    if sub == nil {
       return nil,errors.New("the pro {{$v.PName}} is nil")
    }
    {{- end}}
    sub1 := m.{{UperFirstChar $.MainKeyName}}
    {{$mType := (DetectBaseType $.MainKeyType) -}}
    {{- if not $mType -}} 
    if sub1 == nil {
       return nil,errors.New("the mainkey {{$.MainKeyName}} is nil")
    }
    {{- end}}
    kList := []interface{}{pre,sub,sub1}
    kBuf,cErr := kope.EncodeSlice(kList)
    return kBuf,cErr
}

{{if or (eq $v.SType 1) (eq $v.SType 3) -}}
//Query srt by order 
//
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
//
//f: callback for each traversal , primary 、sub key、idx(the number of times it has been iterated) 
//as arguments to the callback function 
//if the return value of f is true,continue iterating until the end iteration;
//otherwise stop iteration immediately
//
//lastMainKey: the main key of the last one of last page
//lastSubVal: the value  of the last one of last page
//
func (s *S{{$.ClsName}}{{$v.PName}}Wrap) ForEachByOrder(start *{{$v.PType}}, end *{{$v.PType}}, lastMainKey *{{formatStr $.MainKeyType}},
     lastSubVal *{{formatSliceType $v.PType}},f func(mVal *{{formatStr $.MainKeyType}},sVal *{{formatSliceType $v.PType}},idx uint32) bool ) error {
    if s.Dba == nil {
       return errors.New("the db is nil")
    }
    if (lastSubVal != nil && lastMainKey == nil ) || (lastSubVal == nil && lastMainKey != nil) {
       return errors.New("last query param error")
    }
    if f == nil {
       return nil
    }
    pre := {{$.ClsName}}{{$v.PName}}Table
    skeyList := []interface{}{pre}
    if start != nil {
       skeyList = append(skeyList,start)
       if lastMainKey != nil {
          skeyList = append(skeyList,lastMainKey,kope.MinimalKey)
       }
    }else {
        if lastMainKey != nil && lastSubVal != nil {
            skeyList = append(skeyList,lastSubVal,lastMainKey,kope.MinimalKey)
        }
        skeyList = append(skeyList,kope.MinimalKey)
    }
    sBuf,cErr := kope.EncodeSlice(skeyList)
    if cErr != nil {
         return cErr
    }
	eKeyList := []interface{}{pre}
    if end != nil {
       eKeyList = append(eKeyList,end)
    } else {
       eKeyList = append(eKeyList, kope.MaximumKey)
	}
    eBuf,cErr := kope.EncodeSlice(eKeyList)
    if cErr != nil {
       return cErr
    }
    var idx uint32 = 0
	s.Dba.Iterate(sBuf, eBuf, false, func(key, value []byte) bool {
		idx ++
		return f(s.GetMainVal(value), s.GetSubVal(value), idx)
	})
	return nil
}
{{end}}
{{if or (eq $v.SType 2) (eq $v.SType 3) -}}
//Query srt by reverse order 
//
//f: callback for each traversal , primary 、sub key、idx(the number of times it has been iterated) 
//as arguments to the callback function 
//if the return value of f is true,continue iterating until the end iteration;
//otherwise stop iteration immediately
//
//lastMainKey: the main key of the last one of last page
//lastSubVal: the value  of the last one of last page
//
func (s *S{{$.ClsName}}{{$v.PName}}Wrap) ForEachByRevOrder(start *{{$v.PType}}, end *{{$v.PType}},lastMainKey *{{formatStr $.MainKeyType}},
     lastSubVal *{{formatSliceType $v.PType}}, f func(mVal *{{formatStr $.MainKeyType}},sVal *{{formatSliceType $v.PType}}, idx uint32) bool) error {
    if s.Dba == nil {
       return errors.New("the db is nil")
    }
    if (lastSubVal != nil && lastMainKey == nil ) || (lastSubVal == nil && lastMainKey != nil) {
       return errors.New("last query param error")
    }
    if f == nil {
       return nil
    }
    pre := {{$.ClsName}}{{$v.PName}}Table
    skeyList := []interface{}{pre}
    if start != nil {
       skeyList = append(skeyList,start)
       if lastMainKey != nil {
            skeyList = append(skeyList,lastMainKey)
       }
    } else {
       if lastMainKey != nil && lastSubVal != nil {
           skeyList = append(skeyList,lastSubVal,lastMainKey)
       } 
       skeyList = append(skeyList, kope.MaximumKey)
	}
    sBuf,cErr := kope.EncodeSlice(skeyList)
    if cErr != nil {
         return cErr
    }
    eKeyList := []interface{}{pre}
    if end != nil {
       eKeyList = append(eKeyList,end)
    }
    eBuf,cErr := kope.EncodeSlice(eKeyList)
    if cErr != nil {
       return cErr
    }
	var idx uint32 = 0
	s.Dba.Iterate(eBuf, sBuf, true, func(key, value []byte) bool {
		idx ++
		return f(s.GetMainVal(value), s.GetSubVal(value), idx)
	})
	return nil
}
{{end -}}
{{end -}}

/////////////// SECTION Private function ////////////////

func (s *So{{$.ClsName}}Wrap) update(sa *So{{$.ClsName}}) bool {
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

func (s *So{{$.ClsName}}Wrap) get{{$.ClsName}}() *So{{$.ClsName}} {
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

	res := &So{{$.ClsName}}{}
	if proto.Unmarshal(resBuf, res) != nil {
		return nil
	}
	return res
}


func (s *So{{$.ClsName}}Wrap) update{{$.ClsName}}(so *So{{$.ClsName}}) error {
     if s.dba == nil {
		return errors.New("update fail:the db is nil")
	}
	
	if so == nil {
		return errors.New("update fail: the So{{$.ClsName}} is nil")
	}
     
     key, err := s.encodeMainKey()
	if err != nil {
		return nil
	}
	
	buf,err := proto.Marshal(so)
	if err != nil {
		return err
	}
	
	err = s.dba.Put(key, buf)
	if err != nil {
		return err
	}

	return nil
}


func (s *So{{$.ClsName}}Wrap) encodeMainKey() ([]byte, error) {
    if s.mKeyBuf != nil {
		return s.mKeyBuf,nil
	}
    pre := {{$.ClsName}}{{.MainKeyName}}Row
    sub := s.mainKey
    if sub == nil {
       return nil,errors.New("the mainKey is nil")
    }
    preBuf,err := kope.Encode(pre)
	if err != nil {
		return nil,err
	}
    mBuf,err := s.getMainKeyBuf()
    if err != nil {
       return nil,err
    }
	list := make([][]byte,2)
	list[0] = preBuf
	list[1] = mBuf
	s.mKeyBuf = kope.PackList(list)
	return s.mKeyBuf, nil
}

////////////// Unique Query delete/insert/query ///////////////

{{if ge (getMapCount .UniqueFieldMap) 0}}
func (s *So{{$.ClsName}}Wrap)delAllUniKeys(br bool, val *So{{.ClsName}}) bool {
     if s.dba == nil {
       return false
     }
     res := true
     {{range $k, $v := .UniqueFieldMap -}}
	 if !s.delUniKey{{$k}}(val) {
        if br {
           return false
        }else {
           res = false
        }
	 }
	 {{end}}
     return res
}

func (s *So{{$.ClsName}}Wrap)delUniKeysWithNames(names map[string]string, val *So{{.ClsName}}) bool {
     if s.dba == nil {
       return false
     }
     res := true
     {{range $k, $v := .UniqueFieldMap -}}
	 if len(names["{{$k}}"]) > 0 {
        if !s.delUniKey{{$k}}(val) {
           res = false
        }
	 }
	 {{end}}
     return res
}

func (s *So{{$.ClsName}}Wrap)insertAllUniKeys(val *So{{$.ClsName}}) (map[string]string, error) {
     if s.dba == nil {
       return nil,errors.New("insert uniuqe Field fail,the db is nil ")
    }
	if val == nil {
		return nil,errors.New("insert uniuqe Field fail,get the So{{.ClsName}} fail ")
	}
    sucFields := map[string]string{}
    {{range $k, $v := .UniqueFieldMap -}}
	if !s.insertUniKey{{$k}}(val) {
		return sucFields,errors.New("insert unique Field {{$k}} fail while insert table ")
	}
    sucFields["{{$k}}"] = "{{$k}}"
	{{end}}
    return sucFields,nil
}

{{end}}

{{range $k, $v := .UniqueFieldMap}}
func (s *So{{$.ClsName}}Wrap) delUniKey{{$k}}(sa *So{{$.ClsName}}) bool {
    if s.dba == nil {
       return false
    }
    pre := {{$.ClsName}}{{$k}}UniTable
    kList := []interface{}{pre}
    if sa != nil {
       {{ $baseType := (DetectBaseType $v.PType) -}}
       {{if not $baseType -}} 
       if sa.{{UperFirstChar $k}} == nil {
          return true
       }
       {{end}}   
       sub := sa.{{UperFirstChar $k}}
       kList = append(kList,sub)
    }else {
       sub := s.Get{{$k}}()
       {{ $baseType := (DetectBaseType $v.PType) -}}
       {{if not $baseType -}} 
       if sub == nil {
          return true
       }
       {{end}}   
       kList = append(kList,sub)
       
    }
    kBuf,err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *So{{$.ClsName}}Wrap) insertUniKey{{$k}}(sa *So{{$.ClsName}}) bool {
    if s.dba == nil || sa == nil{
       return false
    }
    {{$baseType := (DetectBaseType $v.PType) -}}
    {{- if not $baseType -}}
    if sa.{{$v.PName}} == nil {
       return true
    }
    {{ end -}}    

    pre := {{$.ClsName}}{{$k}}UniTable
    sub := sa.{{UperFirstChar $k}}
    kList := []interface{}{pre,sub}
    kBuf,err := kope.EncodeSlice(kList)
	if err != nil {
		return false
	}
    res,err := s.dba.Has(kBuf)
    if err == nil && res == true {
       //the unique key is already exist
		return false
    }
    val := SoUnique{{$.ClsName}}By{{$k}}{}
    {{if ne $.MainKeyName $k -}}
   	val.{{UperFirstChar $.MainKeyName}} = sa.{{UperFirstChar $.MainKeyName}}
    {{end -}}
	val.{{UperFirstChar $k}} = sa.{{UperFirstChar $k}}
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	return s.dba.Put(kBuf, buf) == nil

}

type Uni{{$.ClsName}}{{$k}}Wrap struct {
	Dba iservices.IDatabaseRW
}

func NewUni{{$.ClsName}}{{$k}}Wrap (db iservices.IDatabaseRW) *Uni{{$.ClsName}}{{$k}}Wrap{
     if db == nil {
        return nil
     }
     wrap := Uni{{$.ClsName}}{{$k}}Wrap{Dba:db}
     return &wrap
}

func (s *Uni{{$.ClsName}}{{$k}}Wrap) UniQuery{{$k}}(start *{{formatStr $v.PType}}) *So{{$.ClsName}}Wrap{
    if start == nil || s.Dba == nil {
       return nil
    }
    pre := {{$.ClsName}}{{$k}}UniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := kope.EncodeSlice(kList)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUnique{{$.ClsName}}By{{$k}}{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			{{ $baseType := (DetectBaseType $.MainKeyType) -}}
            {{- if $baseType -}} 
            wrap := NewSo{{$.ClsName}}Wrap(s.Dba,&res.{{UperFirstChar $.MainKeyName}})
            {{- end -}}   
            {{if not $baseType -}} 
            wrap := NewSo{{$.ClsName}}Wrap(s.Dba,res.{{UperFirstChar $.MainKeyName}})
            {{end }}
			return wrap
		}
	}
    return nil
}


{{end}}

`
	fName := TmlFolder + "so_" + tInfo.Name + ".go"
	if fPtr := CreateFile(fName); fPtr != nil {
		funcMapUper := template.FuncMap{"UperFirstChar": UpperFirstChar,
			"formatStr":           formatStr,
			"LowerFirstChar":      LowerFirstChar,
			"DetectBaseType":      DetectBaseType,
			"formatRTypeStr":      formatRTypeStr,
			"formatQueryParamStr": formatQueryParamStr,
			"formatSliceType":     formatSliceType,
			"getMapCount":         getMapCount,
		}
		t := template.New("go_template")
		t = t.Funcs(funcMapUper)
		t.Parse(tmpl)
		t.Execute(fPtr, createParamsFromTableInfo(tInfo))
		cmd := exec.Command("goimports", "-w", fName)
		err := cmd.Run()
		if err != nil {
			panic(fmt.Sprintf("Table %s: auto import package fail,the error is %s", tInfo.Name, err))
		}
		defer fPtr.Close()
		return true, nil
	} else {
		err = errors.New("get file ptr fail")
		log.Println("get file ptr fail")
		return false, err
	}

}

func createParamsFromTableInfo(tInfo TableInfo) Params {
	para := Params{}
	para.ClsName = UpperFirstChar(tInfo.Name)
	para.LKeys = []string{}
	para.LKeyWithType = make(map[string]string)
	para.MemberKeyMap = make(map[string]Field)
	para.UniqueFieldMap = make(map[string]Field)
	para.SortList = make([]SortField, 0)
	for _, v := range tInfo.PList {
		tType := DelDirtyCharacter(v.VarType)
		if tType == "bytes" {
			tType = "[]byte"
		}
		tName := DelDirtyCharacter(v.VarName)
		fName := rValueFormStr(tName)
		fType :=  formatStr(tType)
		if v.BMainKey {
			para.MainKeyName = fName
			para.MainKeyType = fType
		}
		if v.SortType > 0 {
			pre := getFieldPrefix(fName, tInfo.Name, FieldTypeSort)
			para.LKeys = append(para.LKeys, fName)
			para.LKeyWithType[fName] = fType
			para.SortList = append(para.SortList, SortField{
				Field{
					fName,
					fType,
					pre,
				},
				v.SortType,
			})
		}
		if v.BUnique || v.BMainKey {
			pre := getFieldPrefix(fName, tInfo.Name, FieldTypeUni)
			para.UniqueFieldMap[fName] = Field{fName, fType, pre}
		}
		para.MemberKeyMap[fName] =  Field{
			fName,
			fType,
			getFieldPrefix(fName, tInfo.Name, FieldTypeRow),
		}


	}
	para.SListCount = len(para.SortList)
	return para
}

//delete space and tab character in a string
func DelDirtyCharacter(str string) string {
	res := ""
	if len(str) > 0 {
		reg, _ := regexp.Compile("[ \f\n\r\t\v ]+")
		res = reg.ReplaceAllString(str, "")
	}
	return res
}

func getFieldPrefix(fName,tName string,fType FieldType) uint32 {
	if len(fName) < 1 || len(tName) < 1 {
		return 0
	}
	preStr := ""
	if fType == FieldTypeRow {
		preStr = tName + fName + "Row"
	}else if fType == FieldTypeSort {
		preStr = tName + fName + "Table"
	}else if fType == FieldTypeUni {
		preStr = tName + fName + "UniTable"
	}
	if len(preStr) < 1 {
		return 0
	}
	prefix := crc32.ChecksumIEEE([]byte(preStr))
	if  _,ok := prefixMap[prefix]; ok {
		str := fmt.Sprintf("the field %s in table %s will generate the same key as other fields," +
			"please change the filed or table name",fName,tName)
		panic(str)
	}else {
		prefixMap[prefix] = fName
	}
	return prefix
}

/* uppercase first character of string */
func UpperFirstChar(str string) string {
	for i, v := range str {
		return string(unicode.ToUpper(v)) + str[i+1:]
	}
	return str
}

/* lowercase first character of string */
func LowerFirstChar(str string) string {
	for i, v := range str {
		return string(unicode.ToLower(v)) + str[i+1:]
	}
	return str
}

/*  format params of function in pb tool template, remove the "_" meanWhile uppercase words beside "_"*/
func formatStr(str string) string {
	formStr := ""
	if str != "" {
		if strings.Contains(str, ".") {
			arry := strings.Split(str, ".")
			for k, v := range arry {
				if k != 0 {
					formStr += "."
					formStr += ConvertToPbForm(strings.Split(v, "_"))
				} else {
					formStr += v
				}
			}
		} else if strings.Contains(str, "_") {

			formStr = ConvertToPbForm(strings.Split(str, "_"))
		} else {
			formStr = str
		}
	}

	return formStr
}

/* the return value format of Pb struct format(the first Character is upper case) */
func rValueFormStr(str string) string {
	formStr := ""
	if str != "" {
		formStr = ConvertToPbForm(strings.Split(str, "_"))
	}
	return formStr
}

func ConvertToPbForm(arry []string) string {
	formStr := ""
	for _, v := range arry {
		formStr += UpperFirstChar(v)
	}
	return formStr
}

/* detect if is basic data type*/
func DetectBaseType(str string) bool {
	if str != "" && strings.HasPrefix(str, "[]") {
		return true
	}
	switch str {
	case "string":
		return true
	case "uint8":
		return true
	case "uint16":
		return true
	case "uint32":
		return true
	case "uint64":
		return true
	case "int8":
		return true
	case "int16":
		return true
	case "int32":
		return true
	case "int64":
		return true
	case "int":
		return true
	case "float32":
		return true
	case "[]byte":
		return true
	case "byte":
		return true
	case "float64":
		return true
	case "bool":
		return true
	}
	return false
}

/* format the return value type (if the type is not base data type,the type add *)*/
func formatRTypeStr(str string) string {
	if str != "" {
		if strings.HasPrefix(str, "[]") {
			str = formatSliceType(str)
		}
		if !DetectBaseType(str) {
			return "*" + str
		}
	}
	return str
}

/* format the type of query list params to ptr (if the type is base data type,the type add *)*/
func formatQueryParamStr(str string) string {
	if str != "" {
		if DetectBaseType(str) {
			return "*" + str
		}
	}
	return str
}

func formatSliceType(str string) string {
	if strings.HasPrefix(str, "[]") {
		s := strings.TrimPrefix(str, "[]")
		if !DetectBaseType(s) {
			str = "[]" + "*" + s
		}
	}
	return str
}

func getMapCount(m interface{}) int {
	if reflect.TypeOf(m).Kind() == reflect.Map {
		return len(reflect.ValueOf(m).MapKeys())
	}
	return 0
}

