package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"text/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"unicode"
)

type TableInfo struct {
	Name  string
	PList []PropList
}

type PropList struct {
	VarType  string
	VarName  string
	BMainKey bool
	BUnique  bool
	SortType int //0 not support sort 1:support order 2:support reverse order 3:support order and reverse order
	Index    uint32
	impPath  string //the import path of pb type
}

var TmlFolder = "./app/table/"

func (p *PropList) ToString() string {
	s := ""
	if checkIsSliceType(p.VarType) {
		s = fmt.Sprintf("\trepeated  \t%s\t%s = %d;\n", formatPbSliceType(p.VarType), p.VarName, p.Index)
	}else {
		s = fmt.Sprintf("\t%s\t%s = %d;\n", p.VarType, p.VarName, p.Index)
	}
	return s
}

func (p *PropList) Parse(info []string, index uint32) bool {
	pCount := len(info)
	if pCount < 6 {
		log.Printf("the column of csv is not right")
		return false
	}
	name := info[1]
	if CheckUpperLetter(name) {
		//the field name can't contain uppercase letters
		log.Printf("the field %s contain upper letters \n", name)
		return false
	}
	p.VarType = info[0]
	if strings.HasPrefix(p.VarType,"[]byte")  {
		p.VarType = strings.Replace(p.VarType,"[]byte", "bytes",-1)
	}
	if strings.HasPrefix(p.VarType,"int") || strings.HasPrefix(p.VarType,"uint") {
		p.VarType = conToInt32Str(p.VarType)
	}
	p.VarName = name
	res, err := strconv.ParseBool(strings.Replace(info[2], " ", "", -1))
	if err != nil {
		return false
	}
	p.BMainKey = res

	resUni, errUni := strconv.ParseBool(strings.Replace(info[3], " ", "", -1))
	if errUni != nil {
		return false
	}
	p.BUnique = resUni

	resSort, errSort := strconv.ParseBool(strings.Replace(info[4], " ", "", -1))
	if errSort != nil {
		return false
	}
	resRevSort, errRevSort := strconv.ParseBool(strings.Replace(info[5], " ", "", -1))
	if errRevSort != nil {
		return false
	}
	p.SortType = 0
	if resSort && resRevSort {
		//support sort by order and reverse order
		p.SortType = 3
	} else if resSort {
		//support sort by order
		p.SortType = 1
	} else if resRevSort {
		//support sort by reverse order
		p.SortType = 2
	}

	p.Index = index

	if len(info) > 6 && len(info[6]) > 0 {
		impPath := strings.Replace(info[6]," ","",-1)
		if len(impPath) > 0 {
			p.impPath = impPath
		}
	}

	if index == 1 && !p.BMainKey {
		return false
	}
	if index > 1 && p.BMainKey {
		return false
	}

	return true
}

func ExtractPropListToPB(pl []PropList, name string) string {
	result := fmt.Sprintf("message %s {\n", name)

	for _, val := range pl {
		result = fmt.Sprintf("%s%s", result, val.ToString())
	}

	result = fmt.Sprintf("%s}\n", result)

	return result
}

func ExtractPropListToGoFile(pl []PropList, name string) string {
	result := fmt.Sprintf("message %s {\n", name)

	for _, val := range pl {
		result = fmt.Sprintf("%s%s", result, val.ToString())
	}

	result = fmt.Sprintf("%s}\n", result)

	return result
}

func ProcessCSVFile(fileName string, name string) bool {
	inBuff, err := ioutil.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	r2 := csv.NewReader(strings.NewReader(string(inBuff)))
	lines, _ := r2.ReadAll()

	var indexPb uint32 = 0
	sz := len(lines)
	props := make([]PropList, 0)

	for i := 1; i < sz; i++ {
		line := lines[i]

		if len(line[0]) <= 0 {
			continue
		}
		if indexPb == 0 {
			indexPb = 1
		}

		pList := &PropList{}

		if !pList.Parse(line, indexPb) {

			fmt.Println("parse line error:", line)
			panic(nil)
		}
		indexPb++
		props = append(props, *pList)
	}

	tInfo := TableInfo{name, props}

	wRes, _ := WritePbTplToFile(tInfo)
	if wRes {
		//auto create pb.go file
		cmd := exec.Command("protoc", "-I./",
			"-I./../../../",
			"--go_out=paths=source_relative:.",
			TmlFolder+"so_"+name+".proto")
		err := cmd.Run()
		if err == nil {
			//create detail go file (include update insert delete functions)
			cRes, cErr := CreateGoFile(tInfo)
			if !cRes {
				panic(fmt.Sprintf("create go file fail,name prefix is %s , error is %s",tInfo.Name,cErr))
			}
		} else {
			panic(fmt.Sprintf("create %s pb file fail,cmd error is %s", name, err))
		}
	}else {
		panic(fmt.Sprintf("create so_%s.proto fail",name))
	}

	return true
}

func main() {
	var dirName string = "./app/table/table-define"
	pthSep := string(os.PathSeparator)
	fmt.Println(pthSep)
	files, _ := ioutil.ReadDir(dirName)
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		if strings.HasSuffix(f.Name(), ".csv") {
			ProcessCSVFile(dirName+pthSep+f.Name(), f.Name()[0:len(f.Name())-4])
		}
	}
}

/* writte tmplate content into proto file */
func WritePbTplToFile(tInfo TableInfo) (bool, error) {
	var err error = nil
	if tpl := createPbTpl(tInfo); tpl != "" {
		if isExist, _ := JudgeFileIsExist(TmlFolder); !isExist {
			//folder is not exist,create new folder
			if err := os.Mkdir(TmlFolder, os.ModePerm); err != nil {
				fmt.Printf("create folder fail,the error is:%s \n", err)
				return false, err
			}
		}
		fName := TmlFolder + "so_" + tInfo.Name + ".proto"
		if fPtr := CreateFile(fName); fPtr != nil {
			t := template.New("layout.html")
			funcMap := template.FuncMap{
				"checkIsSliceType":checkIsSliceType,
				"formatPbSliceType":formatPbSliceType,
			    "getPbImpPaths":getPbImpPaths}
			t = t.Funcs(funcMap)
			t.Parse(tpl)
			t.Execute(fPtr, tInfo)
			defer fPtr.Close()
			return true, nil
		} else {
			err = errors.New("get file ptr fail")
			fmt.Println("get file ptr fail")
		}
	} else {
		err = errors.New("create tpl fail")
		fmt.Println("create tpl fail")
	}

	return false, err
}

/* create pb template */
func createPbTpl(t TableInfo) string {
	if len(t.PList) > 0 {
		tpl := ""
		tpl = `
syntax = "proto3";

package table;

option go_package = "github.com/coschain/contentos-go/table";

{{$path := (getPbImpPaths .PList) -}}
{{if ne $path "" -}}
{{$path}}
{{end}}
message so_{{.Name}} {
	{{range $k,$v := .PList}}
    {{- if (checkIsSliceType .VarType) -}}
    repeated                      {{formatPbSliceType .VarType}}   {{.VarName}}     =      {{.Index}};
    {{else -}}
    {{.VarType}}   {{.VarName}}     =      {{.Index}};
    {{end -}}
	{{end}}  
}
`
		tpl = fmt.Sprintf("%s%s", tpl, createKeyTpl(t))
		return tpl
	}
	return ""
}

func createKeyTpl(t TableInfo) string {
	tpl := ""
	if len(t.PList) > 0 {
		var sortList = make([]PropList, 0)
		var uniList = make([]PropList, 0)
		mKeyType, mKeyName := "", ""
		for _, v := range t.PList {
			if v.BMainKey {
				mKeyType = v.VarType
				mKeyName = v.VarName
			}
			if v.SortType > 0 {
				sortList = append(sortList, v)
			}
			if v.BUnique || v.BMainKey {
				uniList = append(uniList, v)
			}
		}
		mKeyPro := PropList{VarName: mKeyName, VarType: mKeyType}
		if len(sortList) > 0 && mKeyType != "" && mKeyName != "" {
			for _, v := range sortList {
				tempTpl := ""
				msgName := fmt.Sprintf("\nmessage so_list_%s_by_%s {\n",
					strings.Replace(t.Name, " ", "", -1),
					strings.Replace(v.VarName, " ", "", -1))
				//tempTpl := creSubTabMsgTpl(v, msgName, mKeyPro)
				if !v.BMainKey {
					tempTpl = creSubTabMsgTpl(v, msgName, mKeyPro)
				} else {
					tempTpl = creSubTabMsgTpl(v, msgName, PropList{})
				}
				if tempTpl != "" {
					tpl += tempTpl
				}
			}
		}

		if len(uniList) > 0 && mKeyType != "" && mKeyName != "" {
			for _, v := range uniList {
				tempTpl := ""
				msgName := fmt.Sprintf("\nmessage so_unique_%s_by_%s {\n",
					strings.Replace(t.Name, " ", "", -1),
					strings.Replace(v.VarName, " ", "", -1))
				if !v.BMainKey {
					tempTpl = creSubTabMsgTpl(v, msgName, mKeyPro)
				} else {
					tempTpl = creSubTabMsgTpl(v, msgName, PropList{})
				}
				if tempTpl != "" {
					tpl += tempTpl
				}
			}
		}
	}
	return tpl
}

func creSubTabMsgTpl(pro PropList, msgName string, mKeyPro PropList) string {
	tpl := ""
	if msgName != "" {
		pro.Index = 1
		tpl = fmt.Sprintf("\n%s", msgName)
		tpl = fmt.Sprintf("%s%s", tpl, pro.ToString())
		if mKeyPro.VarType != "" && mKeyPro.VarName != "" {
			mKeyPro.Index = 2
			tpl = fmt.Sprintf("%s%s", tpl, mKeyPro.ToString())
		}
		tpl += "}\n"
	}

	return tpl
}

/* create detail file */
func CreateFile(fileName string) *os.File {
	var fPtr *os.File
	isExist, _ := JudgeFileIsExist(fileName)
	if !isExist {
		//create file
		if f, err := os.Create(fileName); err != nil {
			log.Printf("create detail go file fail,error:%s\n", err)
		} else {
			fPtr = f
		}
	} else {
		//rewrite the file
		if f, _ := os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm); f != nil {
			fPtr = f
		}
	}

	if fPtr == nil {
		log.Fatal("File pointer is empty \n")

	}
	return fPtr

}

/* judge if the file exists */
func JudgeFileIsExist(fPath string) (bool, error) {
	if fPath == "" {
		return false, errors.New("the file path is empty")
	}
	_, err := os.Stat(fPath)
	if err == nil {
		return true, nil
	} else if !os.IsNotExist(err) {
		return true, err
	}
	return false, err
}

/* check the string is contain Upper letters */
func CheckUpperLetter(str string) bool {
	if str != "" {
		for _, v := range str {
			if unicode.IsUpper(v) {
				return true
			}
		}
	}
	return false
}

func conToInt32Str(str string) string {
	if strings.HasPrefix(str,"int") || strings.HasPrefix(str,"uint") {
		tmpStr := strings.Replace(str, " ","",-1)
		reStr := ""
		t32 := "int32"
		t64 := "int64"
		if strings.HasPrefix(str,"uint") {
			t32,t64 = "uint32","uint64"
		}
		switch tmpStr {
		  case t32:
			  reStr = ""
		  case t64:
			  reStr = ""
		default:
			reStr = t32
		}
		if reStr != "" {
			str = strings.Replace(str,tmpStr,reStr,-1)
		}
	}
	return str
}

func checkIsSliceType(str string) bool {
	if str != "" && strings.HasPrefix(str,"[]"){
		return true
	}
	return false
}

func formatPbSliceType(str string) string {
	if str != "" && strings.HasPrefix(str,"[]") {
		return strings.Replace(str,"[]","",-1)
	}
	return str
}

func getPbImpPaths(pList []PropList) string  {
	 res := ""
	 if len(pList) > 0 {
		 pMap := make(map[string]string)
		 for _,v := range pList {
		 	 if len(v.impPath) > 0 {
				 pMap[v.impPath] = v.impPath
			 }
		 }
		 count := len(pMap)
		 if count > 0 {
			i := 0
			for _,v := range pMap {
				if i != 0 {
					res += "\n"
				}
				res += "import " + "\"" + v + "\"" + ";"
				i++
			}
		 }
	 }

     return res
}