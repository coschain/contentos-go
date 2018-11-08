package main

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"html/template"
	"log"
	"os/exec"
	"reflect"
	"strings"
	"unicode"
)

var tbMask uint64 = 1

type Params struct {
	ClsName 			string
	MainKeyType			string
	MainKeyName			string

	LKeys				[]string
	MemberKeyMap		map[string]string
	LKeyWithType		map[string]string
	UniqueFieldMap      map[string]string
	TBMask				string

}

func CreateGoFile(tIfno TableInfo) (bool,error) {
	var err error = nil
	if tIfno.Name == "" {
		err = errors.New("table name is empty")
		return false,err
	}else if len(tIfno.PList) < 1{
		err = errors.New("table datas are empty")
		return false,err
	}

	tmpl := `

package table

import (
	"github.com/coschain/contentos-go/common/encoding"
	"github.com/coschain/contentos-go/db/storage"
     "github.com/coschain/contentos-go/common/prototype"
	 "github.com/gogo/protobuf/proto"
)

////////////// SECTION Prefix Mark ///////////////
var (
	{{.ClsName}}Table        = []byte("{{.ClsName}}Table")
{{range $k, $v := .LKeys}}
	{{$.ClsName}}{{rValueFormStr $v}}Table = []byte("{{$.ClsName}}{{rValueFormStr $v}}Table")
{{end}}
{{range $k, $v := .UniqueFieldMap}}
	{{$.ClsName}}{{rValueFormStr $k}}Table = []byte("{{$.ClsName}}{{rValueFormStr $k}}Table")
{{end}}
)

////////////// SECTION Wrap Define ///////////////
type So{{.ClsName}}Wrap struct {
	dba 		storage.Database
	mainKey 	*{{formateStr .MainKeyType}}
}

func NewSo{{.ClsName}}Wrap(dba storage.Database, key *{{formateStr .MainKeyType}}) *So{{.ClsName}}Wrap{
	result := &So{{.ClsName}}Wrap{ dba, key}
	return result
}

func (s *So{{.ClsName}}Wrap) CheckExist() bool {
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

func (s *So{{.ClsName}}Wrap) Create{{.ClsName}}(sa *So{{.ClsName}}) bool {

	if sa == nil {
		return false
	}

	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return false
	}

	resBuf, err := proto.Marshal(sa)
	if err != nil {
		return false
	}

	err = s.dba.Put(keyBuf, resBuf)

	if err != nil {
		return false
	}

	// update sort list keys
	{{range $k, $v := .LKeys}}
	if !s.insertSortKey{{$v}}(sa) {
		return false
	}
	{{end}}
  
    //update unique list
    {{range $k, $v := .UniqueFieldMap}}
	if !s.insertUniKey{{rValueFormStr $k}}(sa) {
		return false
	}
	{{end}}
    
	return true
}

////////////// SECTION LKeys delete/insert ///////////////

{{range $k1, $v1 := .LKeys}}
func (s *So{{$.ClsName}}Wrap) delSortKey{{$v1}}(sa *So{{$.ClsName}}) bool {
	val := SoList{{$.ClsName}}By{{$v1}}{}

	val.{{$v1}} = sa.{{$v1}}
	val.{{UperFirstChar $.MainKeyName}} = sa.{{UperFirstChar $.MainKeyName}}

	key, err := encoding.Encode(&val)

	if err != nil {
		return false
	}

	return s.dba.Delete(key) == nil
}


func (s *So{{$.ClsName}}Wrap) insertSortKey{{$v1}}(sa *So{{$.ClsName}}) bool {
	val := SoList{{$.ClsName}}By{{$v1}}{}

	val.{{UperFirstChar $.MainKeyName}} = sa.{{UperFirstChar $.MainKeyName}}
	val.{{UperFirstChar $v1}} = sa.{{UperFirstChar $v1}}

	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	key, err := encoding.Encode(&val)

	if err != nil {
		return false
	}
	return s.dba.Put(key, buf) == nil

}

{{end}}
////////////// SECTION LKeys delete/insert //////////////

func (s *So{{.ClsName}}Wrap) Remove{{.ClsName}}() bool {

	sa := s.get{{.ClsName}}()

	if sa == nil {
		return false
	}

    //delete sort list key
	{{range $k, $v := .LKeys}}
	if !s.delSortKey{{$v}}(sa) {
		return false
	}
	{{end}}
   
    //delete unique list
    {{range $k, $v := .UniqueFieldMap}}
	if !s.delUniKey{{rValueFormStr $k}}(sa) {
		return false
	}
	{{end}}
    
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return false
	}

	return s.dba.Delete(keyBuf) == nil
}

////////////// SECTION Members Get/Modify ///////////////

{{range $k1, $v1 := .MemberKeyMap}}

func (s *So{{$.ClsName}}Wrap) Get{{rValueFormStr $k1}}() *{{formateStr $v1}} {
	res := s.get{{$.ClsName}}()

	if res == nil {
		return nil
	}
{{$baseType := (DetermineBaseType $v1) }}
{{if $baseType}} 
return &res.{{rValueFormStr $k1}}
{{end}}
{{if not $baseType}} 
return res.{{rValueFormStr $k1}}
{{end}}

}

{{if ne $k1 $.MainKeyName}}

func (s *So{{$.ClsName}}Wrap) Md{{rValueFormStr $k1}}(p {{formateStr $v1}}) bool {

	sa := s.get{{$.ClsName}}()

	if sa == nil {
		return false
	}

    {{range $k3, $v3 := $.UniqueFieldMap}}
		{{if eq $k3 $k1 }}
    //judge the unique value is exist
    uniWrap  := SUnique{{$.ClsName}}By{{rValueFormStr $k3}}{}
   {{$baseType := (DetermineBaseType $v3) }}
   {{if $baseType}} 
   	res := uniWrap.UniQuery{{rValueFormStr $k1}}(&sa.{{UperFirstChar $k1}})
   {{end}}
   {{if not $baseType}} 
   	res := uniWrap.UniQuery{{rValueFormStr $k1}}(sa.{{UperFirstChar $k1}})
   {{end}}
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}
	if !s.delUniKey{{rValueFormStr $k3}}(sa) {
		return false
	}
		{{end}}
	{{end}}
    

	{{range $k2, $v2 := $.LKeys}}
		{{if eq $v2 $k1 }}
	if !s.delSortKey{{$k1}}(sa) {
		return false
	}
		{{end}}
	{{end}}

   {{if $baseType}} 
     sa.{{rValueFormStr $k1}} = p
   {{end}}
   {{if not $baseType}} 
     sa.{{rValueFormStr $k1}} = &p
   {{end}}
	
	if !s.update(sa) {
		return false
	}
    {{range $k2, $v2 := $.LKeys}}
      {{if eq $v2 $k1 }}
    if !s.insertSortKey{{$k1}}(sa) {
		return false
    }
       {{end}}
    {{end}}
     
     
    {{range $k3, $v3 := $.UniqueFieldMap}}
		{{if eq $k3 $k1 }}
    if !s.insertUniKey{{rValueFormStr $k3}}(sa) {
		return false
    }
		{{end}}
	{{end}}
	return true
}

{{end}}

{{end}}

{{range $v, $k := .LKeyWithType}}
////////////// SECTION List Keys ///////////////

func (m *SoList{{$.ClsName}}By{{$v}}) OpeEncode() ([]byte, error) {

	mainBuf, err := encoding.Encode(m.{{UperFirstChar $.MainKeyName}})
	if err != nil {
		return nil, err
	}
	subBuf, err := encoding.Encode(m.{{$v}})
	if err != nil {
		return nil, err
	}

	return append(append({{$.ClsName}}{{$v}}Table, subBuf...), mainBuf...), nil
}

type SList{{$.ClsName}}By{{$v}} struct {
	Dba storage.Database
}

func (s *SList{{$.ClsName}}By{{$v}}) GetMainVal(iterator storage.Iterator) *{{formateStr $.MainKeyType}} {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoList{{$.ClsName}}By{{$v}}{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

    {{$baseType := (DetermineBaseType $.MainKeyType) }}
   {{if $baseType}} 
     return &res.{{rValueFormStr $.MainKeyName}}
   {{end}}
   {{if not $baseType}} 
   return res.{{rValueFormStr $.MainKeyName}}
   {{end}}

}

func (s *SList{{$.ClsName}}By{{$v}}) GetSubVal(iterator storage.Iterator) *{{formateStr $k}} {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoList{{$.ClsName}}By{{$v}}{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    {{$baseType := (DetermineBaseType $k) }}
   {{if $baseType}} 
     return &res.{{rValueFormStr $v}}
   {{end}}
   {{if not $baseType}} 
   return res.{{rValueFormStr $v}}
   {{end}}
}

func (s *SList{{$.ClsName}}By{{$v}}) DoList(start {{formateStr $k}}, end {{formateStr $k}}) storage.Iterator {

	startBuf, err := encoding.Encode(&start)
	if err != nil {
		return nil
	}

	endBuf, err := encoding.Encode(&end)
	if err != nil {
		return nil
	}

	bufStartkey := append({{$.ClsName}}{{$v}}Table, startBuf...)
	bufEndkey := append({{$.ClsName}}{{$v}}Table, endBuf...)

	iter := s.Dba.NewIterator(bufStartkey, bufEndkey)

	return iter
}

{{end}}

/////////////// SECTION Private function ////////////////

func (s *So{{$.ClsName}}Wrap) update(sa *So{{$.ClsName}}) bool {
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

func (s *So{{$.ClsName}}Wrap) encodeMainKey() ([]byte, error) {
	res, err := encoding.Encode(s.mainKey)

	if err != nil {
		return nil, err
	}

	return append({{.ClsName}}Table, res...), nil
}

////////////// Unique Query delete/insert/query ///////////////
{{range $k, $v := .UniqueFieldMap}}

func (s *So{{$.ClsName}}Wrap) delUniKey{{rValueFormStr $k}}(sa *So{{$.ClsName}}) bool {
	val := SoUnique{{$.ClsName}}By{{rValueFormStr $k}}{}

	val.{{rValueFormStr $k}} = sa.{{rValueFormStr $k}}
	val.{{UperFirstChar $.MainKeyName}} = sa.{{UperFirstChar $.MainKeyName}}

	key, err := encoding.Encode(&val)

	if err != nil {
		return false
	}

	return s.dba.Delete(append({{$.ClsName}}{{rValueFormStr $k}}Table,key...)) == nil
}


func (s *So{{$.ClsName}}Wrap) insertUniKey{{rValueFormStr $k}}(sa *So{{$.ClsName}}) bool {
    uniWrap  := SUnique{{$.ClsName}}By{{rValueFormStr $k}}{}
   {{$baseType := (DetermineBaseType $v) }}
   {{if $baseType}} 
   	res := uniWrap.UniQuery{{rValueFormStr $k}}(&sa.{{UperFirstChar $k}})
   {{end}}
   {{if not $baseType}} 
   	res := uniWrap.UniQuery{{rValueFormStr $k}}(sa.{{UperFirstChar $k}})
   {{end}}
	if res != nil {
		//the unique key is already exist
		return false
	}
 
    val := SoUnique{{$.ClsName}}By{{rValueFormStr $k}}{}

	val.{{UperFirstChar $.MainKeyName}} = sa.{{UperFirstChar $.MainKeyName}}
	val.{{UperFirstChar $k}} = sa.{{UperFirstChar $k}}
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	key, err := encoding.Encode(&val)

	if err != nil {
		return false
	}
	return s.dba.Put(append({{$.ClsName}}{{rValueFormStr $k}}Table,key...), buf) == nil

}

type SUnique{{$.ClsName}}By{{rValueFormStr $k}} struct {
	Dba storage.Database
}

func (s *SUnique{{$.ClsName}}By{{rValueFormStr $k}}) UniQuery{{rValueFormStr $k}}(start *{{formateStr $v}}) *So{{$.ClsName}}Wrap{

   startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}

	bufStartkey := append({{$.ClsName}}{{rValueFormStr $k}}Table, startBuf...)
	bufEndkey := bufStartkey

	iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
    
    val, err := iter.Value()

	if err != nil {
		return nil
	}

	res := &SoUnique{{$.ClsName}}By{{rValueFormStr $k}}{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}
    wrap := NewSo{{$.ClsName}}Wrap(s.Dba,res.{{UperFirstChar $.MainKeyName}})
    
	return wrap	
}

{{end}}

`
	fName := TmlFolder + "so_"+ tIfno.Name + ".go"
	if fPtr := CreateFile(fName); fPtr != nil {
		funcMapUper := template.FuncMap{"UperFirstChar": UperFirstChar,
		"formateStr":formateStr,
		"LowerFirstChar": LowerFirstChar,
		"rValueFormStr":rValueFormStr,
		"convertToStr":convertToStr,
		"DetermineBaseType":DetermineBaseType}
		t := template.New("layout.html")
		t  = t.Funcs(funcMapUper)
		t.Parse(tmpl)
		t.Execute(fPtr,createParamsFromTableInfo(tIfno))
		cmd := exec.Command("goimports", "-w", fName)
		cmd.Start()
		defer fPtr.Close()
		return true,nil
	}else {
		err = errors.New("get file ptr fail")
		log.Println("get file ptr fail")
		return false,err
	}

}

func createParamsFromTableInfo(tInfo TableInfo) Params {
	para := Params{}
	para.ClsName = UperFirstChar(tInfo.Name)
	para.TBMask = fmt.Sprintf("%d",tbMask)
	tbMask ++
	para.LKeys = []string{}
	para.LKeyWithType = make(map[string]string)
	para.MemberKeyMap = make(map[string]string)
	para.UniqueFieldMap = make(map[string]string)
	for _,v := range tInfo.PList {
		fType :=  strings.Replace(v.VarType," ", "", -1)
		fName :=  strings.Replace(v.VarName," ", "", -1)
		if v.BMainKey {
			para.MainKeyName = fName
			para.MainKeyType =  fType
		}else {
			if v.BSort {
				para.LKeys = append(para.LKeys,formateStr(fName))
				para.LKeyWithType[formateStr(fName)] = fType
			}
		}
		if v.BUnique {
            para.UniqueFieldMap[formateStr(fName)] = fType
		}
		para.MemberKeyMap[formateStr(fName)] = fType
	}
	return para
}

/* upercase first character of string */
func UperFirstChar(str string) string {
	for i, v := range str {
		return string(unicode.ToUpper(v)) + str[i+1:]
	}
	return str
}

func LowerFirstChar(str string) string {
	for i, v := range str {
		return string(unicode.ToLower(v)) + str[i+1:]
	}
	return str
}

func formateStr(str string) string  {
	formStr := ""
	if str != "" {
            if strings.Contains(str, ".") {
				arry := strings.Split(str, ".")
            	for k,v := range arry {
					if k != 0 {
						formStr += "."
						formStr += ConvertToPbForm(strings.Split(v, "_"))
					}else {
						formStr +=  v
					}
				}
			}else if strings.Contains(str,"_"){

				formStr = ConvertToPbForm(strings.Split(str, "_"))
			}else {
				formStr = str
			}
		}

	return formStr
}

func rValueFormStr(str string) string {
	formStr := ""
	if str != "" {
		formStr = ConvertToPbForm(strings.Split(str,"_"))
	}
	return formStr
}

func ConvertToPbForm(arry []string) string {
	formStr := ""
	for _,v := range arry {
		formStr += UperFirstChar(v)
	}
	return formStr
}

func JudgeIsPtr(t interface{}) bool {
	if reflect.TypeOf(t).Kind().String() == reflect.Ptr.String() {
		return true
	}
	return false
}

func convertToStr(t interface{}) string {
	res := fmt.Sprintf("%s",t)
	return res
}

func DetermineBaseType(str string) bool {
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
		case "float64":
	}
	return false
}