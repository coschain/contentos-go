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

type SortPro struct {
	PType string
	PName string
	SType int   //1:支持正序 2:支持倒序 3:支持正序和倒序
}


type Params struct {
	ClsName 			string
	MainKeyType			string
	MainKeyName			string

	LKeys				[]string
	MemberKeyMap		map[string]string
	LKeyWithType		map[string]string
	UniqueFieldMap      map[string]string
	TBMask				string
    SortList            []SortPro
	SListCount          int
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
     {{if ge $.SListCount 1 -}}
     "bytes"
     {{end -}}
     "errors"
     "github.com/coschain/contentos-go/common/encoding"
     "github.com/coschain/contentos-go/prototype"
	 "github.com/gogo/protobuf/proto"
     "github.com/coschain/contentos-go/iservices"
)

////////////// SECTION Prefix Mark ///////////////
var (
	{{.ClsName}}Table        = []byte("{{.ClsName}}Table")
    {{range $k, $v := .SortList -}}
    {{$.ClsName}}{{$v.PName}}Table = []byte("{{$.ClsName}}{{$v.PName}}Table")
    {{$.ClsName}}{{$v.PName}}RevOrdTable = []byte("{{$.ClsName}}{{$v.PName}}RevOrdTable")
    {{end -}}
    {{range $k, $v := .UniqueFieldMap -}}
	{{$.ClsName}}{{$k}}UniTable = []byte("{{$.ClsName}}{{$k}}UniTable")
    {{end -}}
)

////////////// SECTION Wrap Define ///////////////
type So{{.ClsName}}Wrap struct {
	dba 		iservices.IDatabaseService
	mainKey 	*{{formatStr .MainKeyType}}
}

func NewSo{{.ClsName}}Wrap(dba iservices.IDatabaseService, key *{{formatStr .MainKeyType}}) *So{{.ClsName}}Wrap{
	result := &So{{.ClsName}}Wrap{ dba, key}
	return result
}

//return nil if exist
func (s *So{{.ClsName}}Wrap) CheckExist() error {
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return errors.New("encode the mainKey fail")
	}

	res, err := s.dba.Has(keyBuf)
	if err != nil {
		return err
	}
    if !res {
       return errors.New("the table is already exist")
    }
	return nil
}

func (s *So{{.ClsName}}Wrap) Create{{.ClsName}}(f func(t *So{{.ClsName}})) error {

	val := &So{{.ClsName}}{}
    f(val)
    {{$baseType := (DetectBaseType $.MainKeyType) -}}
    {{- if not $baseType -}} 
    if val.{{$.MainKeyName}} == nil {
       return errors.New("the mainkey is nil")
    }
    {{ end -}}
    if s.CheckExist() == nil {
       return errors.New("the mainkey is already exist")
    }
	keyBuf, err := s.encodeMainKey()

	if err != nil {
		return err
	}
	resBuf, err := proto.Marshal(val)
	if err != nil {
		return err
	}
	err = s.dba.Put(keyBuf, resBuf)
	if err != nil {
		return err
	}

	// update sort list keys
	{{range $k, $v := .LKeys}}
	if !s.insertSortKey{{$v}}(val) {
		return errors.New("insert sort Field {{$v}} while insert table ")
	}
	{{end}}
  
    //update unique list
    {{range $k, $v := .UniqueFieldMap -}}
	if !s.insertUniKey{{$k}}(val) {
		return errors.New("insert unique Field {{$v}} while insert table ")
	}
	{{end}}
    
	return nil
}

////////////// SECTION LKeys delete/insert ///////////////
{{range $k1, $v1 := .SortList}}
func (s *So{{$.ClsName}}Wrap) delSortKey{{$v1.PName}}(sa *So{{$.ClsName}}) bool {
	val := SoList{{$.ClsName}}By{{$v1.PName}}{}
	val.{{$v1.PName}} = sa.{{$v1.PName}}
	val.{{UperFirstChar $.MainKeyName}} = sa.{{UperFirstChar $.MainKeyName}}
    {{if eq $v1.SType 1 -}}
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordErr :=  s.dba.Delete(subBuf)
    return ordErr == nil
    {{end -}}
    {{if eq $v1.SType 2 -}}
    subRevBuf, err := val.EncodeRevSortKey()
	if err != nil {
		return false
	}
    revOrdErr :=  s.dba.Delete(subRevBuf) 
    return revOrdErr == nil
    {{end -}}
    {{if eq $v1.SType 3 -}}
    subBuf, err := val.OpeEncode()
    var ordErr,revOrdErr error
	if err == nil {
       ordKey := append({{$.ClsName}}{{$v1.PName}}Table, subBuf...)
       ordErr =  s.dba.Delete(ordKey) 
	}
    subRevBuf, err := val.EncodeRevSortKey()
	if err == nil {
		revOrdKey := append({{$.ClsName}}{{$v1.PName}}RevOrdTable, subRevBuf...)
        revOrdErr =  s.dba.Delete(revOrdKey) 
	}
    if ordErr == nil && revOrdErr == nil {
       return true
    }
    return false
    {{end}}
}


func (s *So{{$.ClsName}}Wrap) insertSortKey{{$v1.PName}}(sa *So{{$.ClsName}}) bool {
	val := SoList{{$.ClsName}}By{{$v1.PName}}{}
    {{if ne $.MainKeyName $v1.PName -}}
   	val.{{UperFirstChar $.MainKeyName}} = sa.{{UperFirstChar $.MainKeyName}}
    {{end -}}
	val.{{UperFirstChar $v1.PName}} = sa.{{UperFirstChar $v1.PName}}
	buf, err := proto.Marshal(&val)
	if err != nil {
		return false
	}
    {{if eq $v1.SType 1 -}}
    subBuf, err := val.OpeEncode()
	if err != nil {
		return false
	}
    ordErr :=  s.dba.Put(subBuf, buf) 
    return ordErr == nil
    {{end -}}
    {{if eq $v1.SType 2 -}}
    subRevBuf, err := val.EncodeRevSortKey()
	if err != nil {
		return false
	}
    revOrdErr :=  s.dba.Put(subRevBuf, buf) 
    return revOrdErr == nil
    {{end -}}
    {{if eq $v1.SType 3 -}}
    subBuf, err := val.OpeEncode()
    var ordErr,revOrdErr error
	if err == nil {
       ordErr =  s.dba.Put(subBuf, buf) 
	}
    subRevBuf, err := val.EncodeRevSortKey()
	if err == nil {
        revOrdErr =  s.dba.Put(subRevBuf, buf) 
	}
    if ordErr == nil && revOrdErr == nil {
       return true
    }
    return false
    {{end}}
}

{{end}}
////////////// SECTION LKeys delete/insert //////////////

func (s *So{{.ClsName}}Wrap) Remove{{.ClsName}}() error {
	sa := s.get{{.ClsName}}()
	if sa == nil {
		return errors.New("delete data fail ")
	}
    //delete sort list key
	{{range $k, $v := .LKeys -}}
	if !s.delSortKey{{$v}}(sa) {
		return errors.New("delete the sort key {{$v}} fail")
	}
	{{end}}
    //delete unique list
    {{range $k, $v := .UniqueFieldMap -}}
	if !s.delUniKey{{$k}}(sa) {
		return errors.New("delete the unique key {{$k}} fail")
	}
	{{end}}
	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return err
	}
    if err := s.dba.Delete(keyBuf); err != nil {
       return err
    }
	return nil
}

////////////// SECTION Members Get/Modify ///////////////
{{range $k1, $v1 := .MemberKeyMap -}}
func (s *So{{$.ClsName}}Wrap) Get{{$k1}}(v *{{formatRTypeStr $v1}}) error {
	res := s.get{{$.ClsName}}()

   if res == nil {
      return errors.New("get table data fail")
   }
   *v =  res.{{$k1}}
   return nil
}

{{if ne $k1 $.MainKeyName}}

func (s *So{{$.ClsName}}Wrap) Md{{$k1}}(p {{formatRTypeStr $v1}}) error {
	sa := s.get{{$.ClsName}}()
	if sa == nil {
		return errors.New("initialization data failed")
	}
    {{- range $k2, $v2 := $.UniqueFieldMap -}}
      {{- if eq $k2 $k1 }}
    //judge the unique value if is exist
    uniWrap  := Uni{{$.ClsName}}{{$k2}}Wrap{}
   {{ $baseType := (DetectBaseType $v2) -}}
   {{- if $baseType -}} 
   	err := uniWrap.UniQuery{{$k1}}(&sa.{{UperFirstChar $k1}},nil)
   {{- end -}}
   {{if not $baseType -}} 
   	err := uniWrap.UniQuery{{$k1}}(sa.{{UperFirstChar $k1}},nil)
   {{end }}
	if err != nil {
		//the unique value to be modified is already exist
		return errors.New("the unique value to be modified is already exist")
	}
	if !s.delUniKey{{$k2}}(sa) {
		return errors.New("delete the unique key {{$k2}} fail")
	}
    {{end -}}
	  {{end}}
	{{range $k3, $v3 := $.LKeys -}}
		{{if eq $v3 $k1 }}
	if !s.delSortKey{{$k1}}(sa) {
		return errors.New("delete the sort key {{$k1}} fail")
	}
		{{- end -}}
	{{end}}
    sa.{{$k1}} = p
	if upErr := s.update(sa);upErr != nil {
		return upErr
	}
    {{range $k4, $v4 := $.LKeys -}}
      {{ if eq $v4 $k1 }}
    if !s.insertSortKey{{$k1}}(sa) {
		return errors.New("reinsert sort key {{$k1}} fail")
    }
       {{end -}}
    {{end}}
    {{- range $k5, $v5 := $.UniqueFieldMap -}}
		{{if eq $k5 $k1 }}
    if !s.insertUniKey{{$k5}}(sa) {
		return errors.New("reinsert unique key {{$k1}} fail")
    }
		{{- end -}}
	{{end}}
	return nil
}
{{end}}
{{end}}

{{range $k, $v := .SortList}}
////////////// SECTION List Keys ///////////////
type S{{$.ClsName}}{{$v.PName}}Wrap struct {
	Dba iservices.IDatabaseService
}

func (s *S{{$.ClsName}}{{$v.PName}}Wrap)DelIterater(iterator iservices.IDatabaseIterator){
   if iterator == nil || !iterator.Valid() {
		return 
	}
   s.Dba.DeleteIterator(iterator)
}

func (s *S{{$.ClsName}}{{$v.PName}}Wrap) GetMainVal(iterator iservices.IDatabaseIterator,mKey *{{formatRTypeStr $.MainKeyType}}) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}
	val, err := iterator.Value()

	if err != nil {
		return err
	}

	res := &SoList{{$.ClsName}}By{{$v.PName}}{}
	err = proto.Unmarshal(val, res)

	if err != nil {
		return err
	}
    *mKey = res.{{$.MainKeyName}}
    return nil
}

func (s *S{{$.ClsName}}{{$v.PName}}Wrap) GetSubVal(iterator iservices.IDatabaseIterator, sub *{{formatRTypeStr $v.PType}}) error {
	if iterator == nil || !iterator.Valid() {
		return errors.New("the iterator is nil or invalid")
	}

	val, err := iterator.Value()

	if err != nil {
		return errors.New("the value of iterator is nil")
	}
	res := &SoList{{$.ClsName}}By{{$v.PName}}{}
	err = proto.Unmarshal(val, res)
	if err != nil {
		return err
	}
    *sub = res.{{ $v.PName}}
    return nil
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
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

func (m *SoList{{$.ClsName}}By{{$v.PName}}) EncodeRevSortKey() ([]byte,error) {
    pre := {{$.ClsName}}{{$v.PName}}RevOrdTable
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
    ordKey,cErr := encoding.EncodeSlice(kList,false)
    if cErr != nil {
       return nil,cErr
    }
    revKey,revRrr := encoding.Complement(ordKey, cErr)
    return revKey,revRrr
}

{{if or (eq $v.SType 1) (eq $v.SType 3) -}}
//Query sort by order 
//start = nil  end = nil (query the db from start to end)
//start = nil (query from start the db)
//end = nil (query to the end of db)
func (s *S{{$.ClsName}}{{$v.PName}}Wrap) QueryListByOrder(start *{{$v.PType}}, end *{{$v.PType}},iter *iservices.IDatabaseIterator) error {
    pre := {{$.ClsName}}{{$v.PName}}Table
    skeyList := []interface{}{pre}
    if start != nil {
       skeyList = append(skeyList,start)
    }
    sBuf,cErr := encoding.EncodeSlice(skeyList,false)
    if cErr != nil {
         return cErr
    }
    if start != nil && end == nil {
		*iter = s.Dba.NewIterator(sBuf, nil)
		return nil
	}
    eKeyList := []interface{}{pre}
    if end != nil {
       eKeyList = append(eKeyList,end)
    }
    eBuf,cErr := encoding.EncodeSlice(eKeyList,false)
    if cErr != nil {
       return cErr
    }
    
    res := bytes.Compare(sBuf,eBuf)
    if res == 0 {
		eBuf = nil
	}else if res == 1 {
       //reverse order
       return errors.New("the start and end are not order")
    }
    *iter = s.Dba.NewIterator(sBuf, eBuf)
    
    return nil
}
{{end}}
{{if or (eq $v.SType 2) (eq $v.SType 3) -}}
//Query sort by reverse order 
func (s *S{{$.ClsName}}{{$v.PName}}Wrap) QueryListByRevOrder(start *{{$v.PType}}, end *{{$v.PType}},iter *iservices.IDatabaseIterator) error {

    var sBuf,eBuf,rBufStart,rBufEnd []byte
    pre := {{$.ClsName}}{{$v.PName}}RevOrdTable
    if start != nil {
       skeyList := []interface{}{pre,start}
       buf,cErr := encoding.EncodeSlice(skeyList,false)
       if cErr != nil {
         return cErr
       }
       sBuf = buf
    }
    
    if end != nil {
       eKeyList := []interface{}{pre,end}
       buf,err := encoding.EncodeSlice(eKeyList,false)
       if err != nil {
          return err
       }
       eBuf = buf

    }

    if sBuf != nil && eBuf != nil {
       res := bytes.Compare(sBuf,eBuf)
       if res == -1 {
          // order
          return errors.New("the start and end are not reverse order")
       }
       if sBuf != nil {
       rBuf,rErr := encoding.Complement(sBuf, nil)
       if rErr != nil {
          return rErr
       }
       rBufStart = rBuf
    }
    if eBuf != nil {
          rBuf,rErr := encoding.Complement(eBuf, nil)
          if rErr != nil { 
            return rErr
          }
          rBufEnd = rBuf
       }
    }
    *iter = s.Dba.NewIterator(rBufStart, rBufEnd)
    return nil
}
{{end -}}
{{end -}}

/////////////// SECTION Private function ////////////////

func (s *So{{$.ClsName}}Wrap) update(sa *So{{$.ClsName}}) error {
	buf, err := proto.Marshal(sa)
	if err != nil {
		return errors.New("initialization data failed")
	}

	keyBuf, err := s.encodeMainKey()
	if err != nil {
		return err
	}
    pErr := s.dba.Put(keyBuf, buf)
    if pErr != nil {
       return pErr
    }
	return nil
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
    pre := {{.ClsName}}Table
    sub := s.mainKey
    if sub == nil {
       return nil,errors.New("the mainKey is nil")
    }
    kList := []interface{}{pre,sub}
    kBuf,cErr := encoding.EncodeSlice(kList,false)
    return kBuf,cErr
}

////////////// Unique Query delete/insert/query ///////////////
{{range $k, $v := .UniqueFieldMap}}

func (s *So{{$.ClsName}}Wrap) delUniKey{{$k}}(sa *So{{$.ClsName}}) bool {
    pre := {{$.ClsName}}{{$k}}UniTable
    sub := sa.{{UperFirstChar $k}}
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Delete(kBuf) == nil
}


func (s *So{{$.ClsName}}Wrap) insertUniKey{{$k}}(sa *So{{$.ClsName}}) bool {
    uniWrap  := Uni{{$.ClsName}}{{$k}}Wrap{}
     uniWrap.Dba = s.dba
   {{$baseType := (DetectBaseType $v) -}}
   {{if $baseType -}} 
   	res := uniWrap.UniQuery{{$k}}(&sa.{{UperFirstChar $k}},nil)
   {{end}}
   {{if not $baseType -}} 
   	res := uniWrap.UniQuery{{$k}}(sa.{{UperFirstChar $k}},nil)
   {{end -}}
	if res == nil {
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
    
    pre := {{$.ClsName}}{{$k}}UniTable
    sub := sa.{{UperFirstChar $k}}
    kList := []interface{}{pre,sub}
    kBuf,err := encoding.EncodeSlice(kList,false)
	if err != nil {
		return false
	}
	return s.dba.Put(kBuf, buf) == nil

}

type Uni{{$.ClsName}}{{$k}}Wrap struct {
	Dba iservices.IDatabaseService
}

func (s *Uni{{$.ClsName}}{{$k}}Wrap) UniQuery{{$k}}(start *{{formatStr $v}},wrap *So{{$.ClsName}}Wrap) error{
    pre := {{$.ClsName}}{{$k}}UniTable
    kList := []interface{}{pre,start}
    bufStartkey,err := encoding.EncodeSlice(kList,false)
    val,err := s.Dba.Get(bufStartkey)
	if err == nil {
		res := &SoUnique{{$.ClsName}}By{{$k}}{}
		rErr := proto.Unmarshal(val, res)
		if rErr == nil {
			{{ $baseType := (DetectBaseType $.MainKeyType) -}}
            {{- if $baseType -}} 
            wrap.mainKey = &res.{{UperFirstChar $.MainKeyName}}
            {{- end -}}   
            {{if not $baseType -}} 
            wrap.mainKey = res.{{UperFirstChar $.MainKeyName}}
            {{end}}
            wrap.dba = s.Dba
			return nil  
		}
        return rErr
	}
    return err
}

{{end}}

`
	fName := TmlFolder + "so_"+ tIfno.Name + ".go"
	if fPtr := CreateFile(fName); fPtr != nil {
		funcMapUper := template.FuncMap{"UperFirstChar": UpperFirstChar,
		"formatStr":formatStr,
		"LowerFirstChar": LowerFirstChar,
		"DetectBaseType":DetectBaseType,
		"formatRTypeStr":formatRTypeStr,
		"formateQueryParamStr":formateQueryParamStr,
		}
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
	para.SortList = make([]SortPro,0)
	for _,v := range tInfo.PList {
		fType :=  strings.Replace(v.VarType," ", "", -1)
		if fType == "bytes" {
			fType = "[]byte"
		}
		fName :=  strings.Replace(v.VarName," ", "", -1)
		if v.BMainKey {
			para.MainKeyName = rValueFormStr(fName)
			para.MainKeyType =  formatStr(fType)
		}
		if v.SortType > 0  {
			para.LKeys = append(para.LKeys,rValueFormStr(fName))
			para.LKeyWithType[rValueFormStr(fName)] = formatStr(fType)
			para.SortList = append(para.SortList,SortPro{
				PName:rValueFormStr(fName),
				PType:formatStr(fType),
				SType:v.SortType,
			})
		}

		if v.BUnique || v.BMainKey {
            para.UniqueFieldMap[rValueFormStr(fName)] = formatStr(fType)
		}
		para.MemberKeyMap[rValueFormStr(fName)] = formatStr(fType)
	}
	para.SListCount = len(para.SortList)
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
	    case "[]byte":
	    	return true
		case "float64":
			 return true
	    case "bool":
		 return true
	}
	return false
}

/* format the return value type (if the type is not base data type,the type add *)*/
func formatRTypeStr(str string) string{
	if str != "" {
		if !DetectBaseType(str) {
			return "*" + str
		}
	}
	return str
}

/* format the type of querylist params to ptr (if the type is base data type,the type add *)*/
func formateQueryParamStr(str string) string {
	if str != "" {
		if DetectBaseType(str) {
			return "*" + str
		}
	}
	return str
}