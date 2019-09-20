package table

import (
	"fmt"
	"github.com/coschain/contentos-go/iservices"
)

type JsonQueryFunc = func(iservices.IDatabaseRW, string)(string, error)

var sTableQueryFuncMap = make(map[string]JsonQueryFunc)

func RegisterTableJsonQuery(tableName string, queryFunc JsonQueryFunc) {
	if queryFunc != nil {
		sTableQueryFuncMap[tableName] = queryFunc
	}
}

func QueryTableRecord(db iservices.IDatabaseRW, tableName string, keyJson string) (valueJson string, err error) {
	if query, ok := sTableQueryFuncMap[tableName]; ok && query != nil {
		valueJson, err = query(db, keyJson)
	} else {
		err = fmt.Errorf("no query function found for table %s", tableName)
	}
	return
}
