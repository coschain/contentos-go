package main

import (
	"fmt"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"html/template"
	"log"
	"os/exec"
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
	{{.ClsName}}Table        = []byte{0x{{$.TBMask}}, 0x0}
{{range $k, $v := .LKeys}}
	{{$.ClsName}}{{$v}}Table = []byte{0x{{$.TBMask}}, 1 + 0x{{$k}} }
{{end}}
)

////////////// SECTION Wrap Define ///////////////
type So{{.ClsName}}Wrap struct {
	dba 		storage.Database
	mainKey 	*{{.MainKeyType}}
}

func NewSo{{.ClsName}}Wrap(dba storage.Database, key *{{.MainKeyType}}) *So{{.ClsName}}Wrap{
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

	// update secondary keys
	{{range $k, $v := .LKeys}}
	if !s.insertSubKey{{$v}}(sa) {
		return false
	}
	{{end}}

	return true
}

////////////// SECTION LKeys delete/insert ///////////////

{{range $k1, $v1 := .LKeys}}
func (s *So{{$.ClsName}}Wrap) deleteSubKey{{$v1}}(sa *So{{$.ClsName}}) bool {
	val := SoKey{{$.ClsName}}By{{$v1}}{}

	val.{{$v1}} = sa.{{$v1}}
	val.{{UperFirstChar $.MainKeyName}} = sa.{{UperFirstChar $.MainKeyName}}

	key, err := encoding.Encode(&val)

	if err != nil {
		return false
	}

	return s.dba.Delete(key) == nil
}


func (s *So{{$.ClsName}}Wrap) insertSubKey{{$v1}}(sa *So{{$.ClsName}}) bool {
	val := SoKey{{$.ClsName}}By{{$v1}}{}

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

	{{range $k, $v := .LKeys}}
	if !s.deleteSubKey{{$v}}(sa) {
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
func (s *So{{$.ClsName}}Wrap) Get{{$.ClsName}}{{formateStr $k1}}() *{{$v1}} {
	res := s.get{{$.ClsName}}()

	if res == nil {
		return nil
	}
	return &res.{{formateStr $k1}}
}


func (s *So{{$.ClsName}}Wrap) Md{{$.ClsName}}{{formateStr $k1}}(p {{$v1}}) bool {

	sa := s.get{{$.ClsName}}()

	if sa == nil {
		return false
	}

	{{range $k2, $v2 := $.LKeys}}
		{{if eq $v2 $k1 }}
	if !s.deleteSubKey{{$k1}}(sa) {
		return false
	}
		{{end}}
	{{end}}
	sa.{{formateStr $k1}} = p
	if !s.update(sa) {
		return false
	}
    {{range $k2, $v2 := $.LKeys}}
      {{if eq $v2 $k1 }}
	   if !s.insertSubKey{{$k1}}(sa) {
		return false
	   }
       {{end}}
    {{end}}
	return true
}

{{end}}


{{range $v, $k := .LKeyWithType}}
////////////// SECTION List Keys ///////////////

func (m *SoKey{{$.ClsName}}By{{$v}}) OpeEncode() ([]byte, error) {

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

func (s *SList{{$.ClsName}}By{{$v}}) GetMainVal(iterator storage.Iterator) *{{$.MainKeyType}} {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoKey{{$.ClsName}}By{{$v}}{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.{{UperFirstChar $.MainKeyName}}
}

func (s *SList{{$.ClsName}}By{{$v}}) GetSubVal(iterator storage.Iterator) *{{$k}} {
	if iterator == nil || !iterator.Valid() {
		return nil
	}

	val, err := iterator.Value()

	if err != nil {
		return nil
	}

	res := &SoKey{{$.ClsName}}By{{$v}}{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return nil
	}

	return &res.{{UperFirstChar $v}}
}

func (s *SList{{$.ClsName}}By{{$v}}) DoList(start {{$k}}, end {{$k}}) storage.Iterator {

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

`
	fName := TmlFolder + "so_"+ tIfno.Name + ".go"
	if fPtr := CreateFile(fName); fPtr != nil {
		funcMapUper := template.FuncMap{"UperFirstChar": UperFirstChar,"formateStr":formateStr}
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
	for _,v := range tInfo.PList {
		fType :=  strings.Replace(v.VarType," ", "", -1)
		fName :=  strings.Replace(v.VarName," ", "", -1)
		if v.BMainKey {
			para.MainKeyName = fName
			para.MainKeyType =  fType
		}else if v.BSeckey {
			para.LKeys = append(para.LKeys,formateStr(fName))
			para.LKeyWithType[formateStr(fName)] = fType
			para.MemberKeyMap[fName] = fType
		}else if v.BUnique {
			para.MemberKeyMap[fName] = fType
		}
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

func formateStr(str string) string  {
	formStr := ""
	if str != "" {
		strArry := strings.Split(str, "_")
		if len(strArry) > 0 {
			for k,_ := range strArry {
				v := strArry[k]
				formStr += UperFirstChar(v)
			}
		}else {
			formStr = UperFirstChar(str)
		}

	}
	return formStr
}
