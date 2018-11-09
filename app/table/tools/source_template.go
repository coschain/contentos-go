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
	{{$.ClsName}}{{$v}}Table = []byte("{{$.ClsName}}{{$v}}Table")
    {{$.ClsName}}{{$v}}RevOrdTable = []byte("{{$.ClsName}}{{$v}}RevOrdTable")
{{end}}
{{range $k, $v := .UniqueFieldMap}}
	{{$.ClsName}}{{$k}}Table = []byte("{{$.ClsName}}{{$k}}Table")
{{end}}
)

////////////// SECTION Wrap Define ///////////////
type So{{.ClsName}}Wrap struct {
	dba 		storage.Database
	mainKey 	*{{formatStr .MainKeyType}}
}

func NewSo{{.ClsName}}Wrap(dba storage.Database, key *{{formatStr .MainKeyType}}) *So{{.ClsName}}Wrap{
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
    {{range $k, $v := .UniqueFieldMap -}}
	if !s.insertUniKey{{$k}}(sa) {
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

    subBuf, err := encoding.Encode(sa.{{UperFirstChar $v1}})
	if err != nil {
		return false
	}
    ordKey := append({{$.ClsName}}{{$v1}}Table, subBuf...)
    revOrdBuf := append({{$.ClsName}}{{$v1}}RevOrdTable, subBuf...)
    revOrdKey,revErr := encoding.Complement(revOrdBuf,err)
    if revErr != nil {
       return false
    }
    ordErr := s.dba.Delete(ordKey)
    revOrdErr := s.dba.Delete(revOrdKey)
    if ordErr == nil && revOrdErr == nil {
        return true 
    }else {
        return false
    }
}


func (s *So{{$.ClsName}}Wrap) insertSortKey{{$v1}}(sa *So{{$.ClsName}}) bool {
	val := SoList{{$.ClsName}}By{{$v1}}{}
	val.{{UperFirstChar $.MainKeyName}} = sa.{{UperFirstChar $.MainKeyName}}
	val.{{UperFirstChar $v1}} = sa.{{UperFirstChar $v1}}
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
    
    subBuf, err := encoding.Encode(sa.{{UperFirstChar $v1}})
	if err != nil {
		return false
	}
    ordKey := append({{$.ClsName}}{{$v1}}Table, subBuf...)
    revOrdBuf := append({{$.ClsName}}{{$v1}}RevOrdTable, subBuf...)
    revOrdKey,revErr := encoding.Complement(revOrdBuf,err)
    if revErr != nil {
       return false
    }
    ordErr :=  s.dba.Put(ordKey, buf) 
    revOrdErr := s.dba.Put(revOrdKey, buf)
    if ordErr == nil && revOrdErr == nil {
        return true 
    }else {
        return false
    }
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
	if !s.delUniKey{{$k}}(sa) {
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

{{range $k1, $v1 := .MemberKeyMap -}}

func (s *So{{$.ClsName}}Wrap) Get{{$k1}}() *{{formatStr $v1}} {
	res := s.get{{$.ClsName}}()
	if res == nil {
		return nil
	}
{{$baseType := (DetectBaseType $v1) -}}
{{if $baseType -}} 
return &res.{{$k1}}
{{end -}}
{{if not $baseType}} 
   return res.{{$k1}}
{{end -}}
}

{{if ne $k1 $.MainKeyName}}

func (s *So{{$.ClsName}}Wrap) Md{{$k1}}(p {{formatStr $v1}}) bool {

	sa := s.get{{$.ClsName}}()

	if sa == nil {
		return false
	}
    {{- range $k2, $v2 := $.UniqueFieldMap -}}
      {{- if eq $k2 $k1 }}
    //judge the unique value if is exist
    uniWrap  := Uni{{$.ClsName}}{{$k2}}Wrap{}
   {{ $baseType := (DetectBaseType $v2) -}}
   {{- if $baseType -}} 
   	res := uniWrap.UniQuery{{$k1}}(&sa.{{UperFirstChar $k1}})
   {{- end -}}
   {{if not $baseType -}} 
   	res := uniWrap.UniQuery{{$k1}}(sa.{{UperFirstChar $k1}})
   {{end }}
	if res != nil {
		//the unique value to be modified is already exist
		return false
	}
	if !s.delUniKey{{$k2}}(sa) {
		return false
	}
    {{end}}
	  {{end}}
	{{range $k3, $v3 := $.LKeys}}
		{{if eq $v3 $k1 }}
	if !s.delSortKey{{$k1}}(sa) {
		return false
	}
		{{end}}
	{{end}}
   {{if $baseType}} 
     sa.{{$k1}} = p
   {{end}}
   {{if not $baseType -}} 
     sa.{{$k1}} = &p
   {{- end}}
	if !s.update(sa) {
		return false
	}
    {{range $k4, $v4 := $.LKeys -}}
      {{- if eq $v4 $k1 }}
    if !s.insertSortKey{{$k1}}(sa) {
		return false
    }
       {{end}}
    {{end}}

    {{range $k5, $v5 := $.UniqueFieldMap}}
		{{if eq $k5 $k1 }}
    if !s.insertUniKey{{$k5}}(sa) {
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

func (m *SoList{{$.ClsName}}By{{$v}}) OpeEncode() ([]byte, []byte,error) {

	mainBuf, err := encoding.Encode(m.{{UperFirstChar $.MainKeyName}})
	if err != nil {
		return nil,nil,err
	}
	subBuf, err := encoding.Encode(m.{{$v}})
	if err != nil {
		return nil,nil,err
	}
   ordKey := append(append({{$.ClsName}}{{$v}}Table, subBuf...), mainBuf...)
   revOrdBuf := append(append({{$.ClsName}}{{$v}}RevOrdTable, subBuf...), mainBuf...)
   revSubKey,revErr := encoding.Complement(revOrdBuf,err)
   if revErr != nil {
      return nil,nil,revErr
   }
   return ordKey,revSubKey,nil
}

type S{{$.ClsName}}{{$v}}Wrap struct {
	Dba storage.Database
}

func (s *S{{$.ClsName}}{{$v}}Wrap) GetMainVal(iterator storage.Iterator) *{{formatStr $.MainKeyType}} {
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

    {{$baseType := (DetectBaseType $.MainKeyType) }}
   {{if $baseType}} 
     return &res.{{$.MainKeyName}}
   {{end}}
   {{if not $baseType}} 
   return res.{{$.MainKeyName}}
   {{end}}

}

func (s *S{{$.ClsName}}{{$v}}Wrap) GetSubVal(iterator storage.Iterator) *{{formatStr $k}} {
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
    {{$baseType := (DetectBaseType $k) }}
   {{if $baseType}} 
     return &res.{{ $v}}
   {{end}}
   {{if not $baseType}} 
   return res.{{$v}}
   {{end}}
}

//Query by sort 
//sort by reverse order: the encoded value of start greater than end 
//sort by order: the encoded value of start less or equal  end 
func (s *S{{$.ClsName}}{{$v}}Wrap) QueryList(start {{formatStr $k}}, end {{formatStr $k}}) storage.Iterator {

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
    if bytes.Compare(startBuf,endBuf) > 1 {
       //reverse order 
       rBufStart,rErr := encoding.Complement(bufStartkey, err)
       if rErr != nil {
       	return nil
	   }
       rBufEnd,rErr := encoding.Complement(bufEndkey, err)
		if rErr != nil {
			return nil
		}
       iter := s.Dba.NewIterator(rBufStart, rBufEnd)
       return iter
    }else {
       iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
		return iter
    }
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

func (s *So{{$.ClsName}}Wrap) delUniKey{{$k}}(sa *So{{$.ClsName}}) bool {
	val := SoUnique{{$.ClsName}}By{{$k}}{}

	val.{{$k}} = sa.{{$k}}
	val.{{UperFirstChar $.MainKeyName}} = sa.{{UperFirstChar $.MainKeyName}}

	key, err := encoding.Encode(sa.{{UperFirstChar $k}})

	if err != nil {
		return false
	}

	return s.dba.Delete(append({{$.ClsName}}{{$k}}Table,key...)) == nil
}


func (s *So{{$.ClsName}}Wrap) insertUniKey{{$k}}(sa *So{{$.ClsName}}) bool {
    uniWrap  := Uni{{$.ClsName}}{{$k}}Wrap{}
   {{$baseType := (DetectBaseType $v) }}
   {{if $baseType}} 
   	res := uniWrap.UniQuery{{$k}}(&sa.{{UperFirstChar $k}})
   {{end}}
   {{if not $baseType}} 
   	res := uniWrap.UniQuery{{$k}}(sa.{{UperFirstChar $k}})
   {{end}}
	if res != nil {
		//the unique key is already exist
		return false
	}
 
    val := SoUnique{{$.ClsName}}By{{$k}}{}

    
	val.{{UperFirstChar $.MainKeyName}} = sa.{{UperFirstChar $.MainKeyName}}
	val.{{UperFirstChar $k}} = sa.{{UperFirstChar $k}}
    
	buf, err := proto.Marshal(&val)

	if err != nil {
		return false
	}

	key, err := encoding.Encode(sa.{{UperFirstChar $k}})

	if err != nil {
		return false
	}
	return s.dba.Put(append({{$.ClsName}}{{$k}}Table,key...), buf) == nil

}

type Uni{{$.ClsName}}{{$k}}Wrap struct {
	Dba storage.Database
}

func (s *Uni{{$.ClsName}}{{$k}}Wrap) UniQuery{{$k}}(start *{{formatStr $v}}) *So{{$.ClsName}}Wrap{

   startBuf, err := encoding.Encode(start)
	if err != nil {
		return nil
	}
	bufStartkey := append({{$.ClsName}}{{$k}}Table, startBuf...)
	bufEndkey := bufStartkey
	iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
    val, err := iter.Value()
	if err != nil {
		return nil
	}
	res := &SoUnique{{$.ClsName}}By{{$k}}{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return nil
	}
   {{ $baseType := (DetectBaseType $.MainKeyType) -}}
   {{- if $baseType -}} 
   wrap := NewSo{{$.ClsName}}Wrap(s.Dba,&res.{{UperFirstChar $.MainKeyName}})
   {{- end -}}
   {{if not $baseType -}} 
   wrap := NewSo{{$.ClsName}}Wrap(s.Dba,res.{{UperFirstChar $.MainKeyName}})
   {{end }}
    
	return wrap	
}

{{end}}

`
	fName := TmlFolder + "so_"+ tIfno.Name + ".go"
	if fPtr := CreateFile(fName); fPtr != nil {
		funcMapUper := template.FuncMap{"UperFirstChar": UpperFirstChar,
		"formatStr":formatStr,
		"LowerFirstChar": LowerFirstChar,
		"DetectBaseType":DetectBaseType}
		t := template.New("layout.html")
		t  = t.Funcs(funcMapUper)
		t.Parse(tmpl)
		t.Execute(fPtr,createParamsFromTableInfo(tIfno))
		cmd := exec.Command("goimports", "-I./","-I./../../../","-w=./", fName)
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
	para.ClsName = UpperFirstChar(tInfo.Name)
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
			para.MainKeyName = rValueFormStr(fName)
			para.MainKeyType =  formatStr(fType)
		}else {
			if v.BSort {
				para.LKeys = append(para.LKeys,rValueFormStr(fName))
				para.LKeyWithType[rValueFormStr(fName)] = formatStr(fType)
			}
		}
		if v.BUnique {
            para.UniqueFieldMap[rValueFormStr(fName)] = formatStr(fType)
		}
		para.MemberKeyMap[rValueFormStr(fName)] = formatStr(fType)
	}
	return para
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

/*  formate params of function in pb tool template, remove the "_" meanWhile uppercase words beside "_"*/
func formatStr(str string) string  {
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

/* the return value format of Pb struct format(the first Charater is upper case) */
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
		formStr += UpperFirstChar(v)
	}
	return formStr
}

/* detect if is basic data type*/
func DetectBaseType(str string) bool {
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