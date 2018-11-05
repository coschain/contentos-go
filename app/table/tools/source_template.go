package main

//func main()  {
//	tmpl := `
//
//package table
//
//import (
//	"github.com/coschain/contentos-go/common/encoding"
//	"github.com/coschain/contentos-go/db/storage"
//	base "github.com/coschain/contentos-go/proto/type-proto"
//	"github.com/gogo/protobuf/proto"
//)
//
//////////////// SECTION Prefix Mark ///////////////
//var (
//	{{.ClsName}}Table        = []byte{0x{{$.TBMask}}, 0x0}
//{{range $k, $v := .LKeys}}
//	{{$v}}Table = []byte{0x{{$.TBMask}}, 1 + 0x{{$k}} }
//{{end}}
//)
//
//////////////// SECTION Wrap Define ///////////////
//type So{{.ClsName}}Wrap struct {
//	dba 		storage.Database
//	mainKey 	*{{.MainKeyType}}
//}
//
//func NewSo{{.ClsName}}Wrap(dba storage.Database, key *{{.MainKeyType}}) *SoAccountWrap{
//	result := &So{{.ClsName}}Wrap{ dba, key}
//	return result
//}
//
//func (s *So{{.ClsName}}Wrap) CheckExist() bool {
//	keyBuf, err := s.encodeMainKey()
//	if err != nil {
//		return false
//	}
//
//	res, err := s.dba.Has(keyBuf)
//	if err != nil {
//		return false
//	}
//
//	return res
//}
//
//func (s *So{{.ClsName}}Wrap) Create{{.ClsName}}(sa *So{{.ClsName}}) bool {
//
//	if sa == nil {
//		return false
//	}
//
//	keyBuf, err := s.encodeMainKey()
//
//	if err != nil {
//		return false
//	}
//
//	resBuf, err := proto.Marshal(sa)
//	if err != nil {
//		return false
//	}
//
//	err = s.dba.Put(keyBuf, resBuf)
//
//	if err != nil {
//		return false
//	}
//
//	// update secondary keys
//	{{range $k, $v := .LKeys}}
//	if !s.insertSubKey{{$v}}(sa) {
//		return false
//	}
//	{{end}}
//
//	return true
//}
//
//////////////// SECTION LKeys delete/insert ///////////////
//
//{{range $k1, $v1 := .LKeys}}
//func (s *So{{$.ClsName}}Wrap) deleteSubKey{{$v1}}(sa *So{{$.ClsName}}) bool {
//	val := SKey{{$.ClsName}}By{{$v1}}{}
//
//	val.{{$v1}} = sa.{{$v1}}
//	val.{{$.MainKeyName}} = sa.{{$.MainKeyName}}
//
//	key, err := encoding.Encode(&val)
//
//	if err != nil {
//		return false
//	}
//
//	return s.dba.Delete(key) == nil
//}
//
//
//func (s *So{{$.ClsName}}Wrap) insertSubKey{{$v1}}(sa *So{{$.ClsName}}) bool {
//	val := SKey{{$.ClsName}}By{{$v1}}{}
//
//	val.{{$.MainKeyName}} = sa.{{$.MainKeyName}}
//	val.{{$v1}} = sa.{{$v1}}
//
//	buf, err := proto.Marshal(&val)
//
//	if err != nil {
//		return false
//	}
//
//	key, err := encoding.Encode(&val)
//
//	if err != nil {
//		return false
//	}
//	return s.dba.Put(key, buf) == nil
//
//}
//
//{{end}}
//
//
//func (s *So{{.ClsName}}Wrap) Remove{{.ClsName}}() bool {
//
//	sa := s.get{{.ClsName}}()
//
//	if sa == nil {
//		return false
//	}
//
//	{{range $k, $v := .LKeys}}
//	if !s.deleteSubKey{{$v}}(sa) {
//		return false
//	}
//	{{end}}
//
//	keyBuf, err := s.encodeMainKey()
//
//	if err != nil {
//		return false
//	}
//
//	return s.dba.Delete(keyBuf) == nil
//}
//
//////////////// SECTION Members Get/Modify ///////////////
//
//{{range $k1, $v1 := .MemberKeyMap}}
//func (s *So{{$.ClsName}}Wrap) Get{{$.ClsName}}{{$k1}}() *{{$v1}} {
//	res := s.get{{$.ClsName}}()
//
//	if res == nil {
//		return nil
//	}
//	return res.{{$k1}}
//}
//
//
//func (s *So{{$.ClsName}}Wrap) Md{{$.ClsName}}{{$k1}}(p {{$v1}}) bool {
//
//	sa := s.get{{$.ClsName}}()
//
//	if sa == nil {
//		return false
//	}
//
//	{{range $k2, $v2 := $.LKeys}}
//		{{if eq $v2 $k1 }}
//	if !s.deleteSubKey{{$k1}}(sa) {
//		return false
//	}
//		{{end}}
//	{{end}}
//	sa.{{$k1}} = &p
//	if !s.update(sa) {
//		return false
//	}
//	{{range $k2, $v2 := $.LKeys}}
//		{{if eq $v2 $k1 }}
//	if !s.insertSubKey{{$k1}}(sa) {
//		return false
//	}
//		{{end}}
//	{{end}}
//	return true
//}
//
//{{end}}
//
//
//{{range $v, $k := .LKeyWithType}}
//////////////// SECTION List Keys ///////////////
//
//func (m *SKey{{$.ClsName}}By{{$v}}) OpeEncode() ([]byte, error) {
//
//	mainBuf, err := encoding.Encode(m.{{$.MainKeyName}})
//	if err != nil {
//		return nil, err
//	}
//	subBuf, err := encoding.Encode(m.{{$v}})
//	if err != nil {
//		return nil, err
//	}
//
//	return append(append({{$v}}Table, subBuf...), mainBuf...), nil
//}
//
//type SList{{$.ClsName}}By{{$v}} struct {
//	Dba storage.Database
//}
//
//func (s *SList{{$.ClsName}}By{{$v}}) GetMainVal(iterator storage.Iterator) *{{$.MainKeyType}} {
//	if iterator == nil || !iterator.Valid() {
//		return nil
//	}
//
//	val, err := iterator.Value()
//
//	if err != nil {
//		return nil
//	}
//
//	res := &SKey{{$.ClsName}}By{{$v}}{}
//	err = proto.Unmarshal(val, res)
//
//	if err != nil {
//		return nil
//	}
//
//	return &res.{{$.MainKeyName}}
//}
//
//func (s *SList{{$.ClsName}}By{{$v}}) GetSubVal(iterator storage.Iterator) *{{$k}} {
//	if iterator == nil || !iterator.Valid() {
//		return nil
//	}
//
//	val, err := iterator.Value()
//
//	if err != nil {
//		return nil
//	}
//
//	res := &SKey{{$.ClsName}}By{{$v}}{}
//	err = proto.Unmarshal(val, res)
//
//	if err != nil {
//		return nil
//	}
//
//	return res.{{$v}}
//}
//
//func (s *SList{{$.ClsName}}By{{$v}}) DoList(start {{$k}}, end {{$k}}) storage.Iterator {
//
//	startBuf, err := encoding.Encode(&start)
//	if err != nil {
//		return nil
//	}
//
//	endBuf, err := encoding.Encode(&end)
//	if err != nil {
//		return nil
//	}
//
//	bufStartkey := append({{$v}}Table, startBuf...)
//	bufEndkey := append({{$v}}Table, endBuf...)
//
//	iter := s.Dba.NewIterator(bufStartkey, bufEndkey)
//
//	return iter
//}
//
//{{end}}
//
///////////////// SECTION Private function ////////////////
//
//func (s *So{{$.ClsName}}Wrap) update(sa *So{{$.ClsName}}) bool {
//	buf, err := proto.Marshal(sa)
//	if err != nil {
//		return false
//	}
//
//	keyBuf, err := s.encodeMainKey()
//	if err != nil {
//		return false
//	}
//
//	return s.dba.Put(keyBuf, buf) == nil
//}
//
//func (s *So{{$.ClsName}}Wrap) get{{$.ClsName}}() *So{{$.ClsName}} {
//	keyBuf, err := s.encodeMainKey()
//
//	if err != nil {
//		return nil
//	}
//
//	resBuf, err := s.dba.Get(keyBuf)
//
//	if err != nil {
//		return nil
//	}
//
//	res := &So{{$.ClsName}}{}
//	if proto.Unmarshal(resBuf, res) != nil {
//		return nil
//	}
//	return res
//}
//
//func (s *So{{$.ClsName}}Wrap) encodeMainKey() ([]byte, error) {
//	res, err := encoding.Encode(s.mainKey)
//
//	if err != nil {
//		return nil, err
//	}
//
//	return append(mainTable, res...), nil
//}
//
//`
//
//	type Params struct {
//		ClsName 			string
//		MainKeyType			string
//		MainKeyName			string
//
//		LKeys				[]string
//		MemberKeyMap		map[string]string
//		LKeyWithType		map[string]string
//		TBMask				string
//
//	}
//
//	fPath := "./tml/"
//	if isExist, _ := JudgeFileIsExist(fPath); !isExist {
//		//文件夹不存在,创建文件夹
//		if err := os.Mkdir(fPath, os.ModePerm); err != nil {
//			log.Fatalf("create folder fail,the error is:%s", err)
//			return
//		}
//	}
//
//	fileName := fPath + "tml1.go"
//
//	var fPtr *os.File
//	isExist, _ := JudgeFileIsExist(fileName)
//	if !isExist {
//		//.go文件不存在,创建文件
//		if f, err := os.Create(fileName); err != nil {
//			log.Fatalf("create detail go file fail,error:%s", err)
//			return
//		} else {
//			fPtr = f
//		}
//	} else {
//		//文件已经存在则重新写入
//		if f, _ := os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm); f != nil {
//			fPtr = f
//		}
//	}
//
//	if fPtr == nil {
//		//获取文件句柄失败
//		log.Fatal("File pointer is empty \n")
//		return
//	}
//	defer fPtr.Close()
//
//	t := template.New("layout.html")
//	t, _ = t.Parse(tmpl)
//	fmt.Println(t.Name())
//
//
//	t.Execute( fPtr, Params{
//		ClsName:"Account",
//		MainKeyType:"base.AccountName",
//		MainKeyName:"Name",
//		TBMask:"1",
//		LKeys:[]string{"CreatedTime", "PubKey"},
//		LKeyWithType:map[string]string {
//			"CreatedTime" :"base.TimePointSec",
//			"PubKey" :"base.PublicKeyType",
//		},
//
//
//		MemberKeyMap:map[string]string {
//			"CreatedTime" :"base.TimePointSec",
//			"PubKey" :"base.PublicKeyType",
//			"Creator" :"base.AccountName",
//		},
//	})
//
//
//	cmd := exec.Command("goimports", "-w", fileName)
//	cmd.Run()
//
//	fmt.Println("\n\n\n\n---------------------------------------------------------\n\n\n\n")
//
//	t.Execute( os.Stdout, Params{
//		ClsName:"Post",
//		MainKeyType:"uint32",
//		MainKeyName:"Idx",
//		TBMask:"2",
//		LKeys:[]string{"Name", "PostTime"},
//		LKeyWithType:map[string]string {
//			"Name" :"base.AccountName",
//			"PostTime" :"base.TimePointSec",
//		},
//
//
//		MemberKeyMap:map[string]string {
//			"Name" :"base.AccountName",
//			"PostTime" :"base.TimePointSec",
//			"Content" :"string",
//			"LikeCount":"uint32",
//		},
//	})
//}

///* 判断文件是否存在 */
//func JudgeFileIsExist(fPath string) (bool, error) {
//	if fPath == "" {
//		return false, errors.New("the file path is empty")
//	}
//	_, err := os.Stat(fPath)
//	if err == nil {
//		return true, nil
//	} else if !os.IsNotExist(err) {
//		return true, err
//	}
//	return false, err
//}

