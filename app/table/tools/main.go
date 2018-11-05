package main

import (
	"encoding/csv"
	"errors"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type TableInfo struct {
	Name 		string
	PList		[]PropList
}


type PropList struct {
	VarType		string
	VarName		string
	BMainKey	bool
	BUnique		bool
	BSort		bool
	Index		uint32
}


func (p *PropList) ToString() string{
	s := fmt.Sprintf("\t%s\t%s = %d;\n", p.VarType, p.VarName, p.Index)
	return s
}

func (p *PropList) Parse(info []string, index uint32) bool {
	return true
	p.VarType	= info[0]
	p.VarName	= info[1]
	res, err	:= strconv.ParseBool(info[2])
	if err != nil{
		return false
	}
	p.BMainKey = res

	res, err	= strconv.ParseBool(info[3])
	if err != nil{
		return false
	}
	p.BUnique	= res

	res, err	= strconv.ParseBool(info[4])
	if err != nil{
		return false
	}
	p.BSort		= res

	p.Index		= index

	if index == 1 && !p.BMainKey{
		return false
	}
	if index > 1 && p.BMainKey{
		return false
	}

	return true
}

func ExtractPropListToPB( pl []PropList, name string) string {
	result := fmt.Sprintf("message %s {\n", name)

	for _, val := range pl {
		result = fmt.Sprintf("%s%s", result, val.ToString() )
	}

	result = fmt.Sprintf("%s}\n", result )

	return result
}

func ExtractPropListToGoFile( pl []PropList, name string) string {
	result := fmt.Sprintf("message %s {\n", name)

	for _, val := range pl {
		result = fmt.Sprintf("%s%s", result, val.ToString() )
	}

	result = fmt.Sprintf("%s}\n", result )

	return result
}

func ProcessCSVFile(fileName string, name string) bool {
	inBuff,err := ioutil.ReadFile(fileName)
	if err != nil {
		panic(err)
	}

	r2 		:= csv.NewReader(strings.NewReader(string(inBuff)))
	lines,_ := r2.ReadAll()

	var indexPb	uint32 = 0
	sz := len(lines)
	props := make([]PropList, 0)

	for i:=1;i<sz;i++ {
		line := lines[i]

		if len(line[0]) <= 0 {
			continue
		}
		if indexPb == 0{
			indexPb = 1
		}

		pList := &PropList{}

		if !pList.Parse( line, indexPb) {

			fmt.Println("parse line error:", line )
			panic( nil )
		}

		indexPb++
		props = append(props, *pList)
	}


	var res = ExtractPropListToPB( props, "so_"+name )
	fmt.Println(res)


	res = ExtractPropListToGoFile( props, "so_"+name )
	fmt.Println(res)


	pList := []PropList{{VarName:"name",VarType:"string",Index:1}}
	tInfo := TableInfo{"so_"+name, pList}


	createPbFile(tInfo)

	return true
}

func main(){
	var dirName string = "./table/table-define"
	pthSep := string(os.PathSeparator)
	fmt.Println(pthSep)
	files, _ := ioutil.ReadDir(dirName)
	for _, f := range files {
		if f.IsDir(){
			continue
		}
		if strings.HasSuffix( f.Name(), ".csv" ){
			ProcessCSVFile(dirName + pthSep + f.Name(), f.Name()[0:len(f.Name())-4 ] )
		}
	}
}


/* 生成pb结构文件 */
func createPbFile(tInfo TableInfo) (bool, error) {
	var err error = nil
	if tpl := createPbTpl(tInfo); tpl != "" {
		if isExist, _ := JudgeFileIsExist("./tml"); !isExist {
			//文件夹不存在,创建文件夹
			if err := os.Mkdir("./tml", os.ModePerm); err != nil {
					log.Fatalf("create folder fail,the error is:%s", err)
			 		return false,err
			 }
			}
		fName := "./tml/" + tInfo.Name + ".proto"
		fmt.Printf("the proto fileName:%s",fName)
		if fPtr := createFile(fName); fPtr != nil {
			tmpName := tInfo.Name + "probo"
			t := template.New(tmpName)
			t.Parse(tpl)
			t.Execute(fPtr,tInfo)
			cmd := exec.Command("goimports", "-w", fName)
			cmd.Run()
			defer fPtr.Close()
			return true,nil
		}else {
			err = errors.New("get file ptr fail")
			log.Fatalf("get file ptr fail")
		}
	}else{
		err = errors.New("create tpl fail")
		log.Fatalf("create tpl fail")
	}

	return false,err
}

/* 根据csv表结构创建pb对应的结构的模板 */
func createPbTpl(t TableInfo) string  {
	if len(t.PList) > 0 {
		return `
syntax = "proto3";

package table;

import "github.com/coschain/contentos-go/proto/type-proto/type.proto";

message {{.Name}} {
	{{range $k,$v := .PList}}
       {{.VarType}}   {{.VarName}}     =      {{.Index}};
	{{end}}  
} 
`
	}
	return ""
}


/* 生成对应的本地文件 */
func createFile(fileName string) *os.File {
	var fPtr *os.File
	isExist, _ := JudgeFileIsExist(fileName)
	if !isExist {
		//.go文件不存在,创建文件
		if f, err := os.Create(fileName); err != nil {
			log.Fatalf("create detail go file fail,error:%s", err)
		} else {
			fPtr = f
		}
	} else {
		//文件已经存在则重新写入
		if f, _ := os.OpenFile(fileName, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, os.ModePerm); f != nil {
			fPtr = f
		}
	}

	if fPtr == nil {
		//获取文件句柄失败
		log.Fatal("File pointer is empty \n")

	}
	return fPtr

}


/* 判断文件是否存在 */
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