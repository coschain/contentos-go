package main

import (
	"encoding/csv"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

type TableInfo struct {
	Name 		string
	PList		PropList
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

	return true
}

func main(){
	var dirName string = "/Users/yykingking/tables/"

	pthSep := string(os.PathSeparator)

	files, _ := ioutil.ReadDir(dirName)
	for _, f := range files {

		if f.IsDir(){
			continue
		}

		fmt.Println(f.Name())
		if strings.HasSuffix( f.Name(), ".csv" ){
			ProcessCSVFile(dirName + pthSep + f.Name(), f.Name()[0:len(f.Name())-4 ] )
		}
	}
}