package table

import (
	"fmt"
	"strings"
	"unicode"
)


type FieldMdHandleType int

const (
	FieldMdHandleTypeCheck  FieldMdHandleType = 0 //check unique field is already exist in table
	FieldMdHandleTypeDel    FieldMdHandleType = 1 //delete sort and unique struct
    FieldMdHandleTypeInsert FieldMdHandleType = 2 //insert sort and unique struct to table
)

func ConvTableFieldToPbFormat(fName string) string {
	res := ""
	if fName != "" {
		sli := strings.Split(fName, "_")
		for _, v := range sli {
			res += UpperFirstChar(v)
		}
	}
	return res
}


/* uppercase first character of string */
func UpperFirstChar(str string) string {
	for i, v := range str {
		return string(unicode.ToUpper(v)) + str[i+1:]
	}
	return str
}

func bindErrorInfo(what interface{}, customs...interface{}) error {
	customMsg := ""
	if len(customs) > 0 {
		customMsg = fmt.Sprint(customs...)
	}
	if len(customMsg) > 0 {
		customMsg += ": "
	}
	return fmt.Errorf("%s%v", customMsg, what)
}
