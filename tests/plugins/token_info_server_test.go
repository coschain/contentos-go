package tests

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/coschain/contentos-go/iservices"
	"github.com/coschain/contentos-go/iservices/itype"
	"reflect"
	"testing"
)

func SetField(obj *itype.ContractData, name string, value interface{}) error {
	structValue := reflect.ValueOf(obj).Elem()
	structFieldValue := structValue.FieldByName(name)

	if !structFieldValue.IsValid() {
		return fmt.Errorf("No such field: %s in obj", name)
	}

	if !structFieldValue.CanSet() {
		return fmt.Errorf("Cannot set %s field value", name)
	}

	structFieldType := structFieldValue.Type()
	val := reflect.ValueOf(value)
	if structFieldType != val.Type() {
		return errors.New("Provided value type didn't match obj field type")
	}

	structFieldValue.Set(val)
	return nil
}

func TestTokenInfoQueryToken(t *testing.T) {
	db, err := sql.Open("mysql", "contentos:123456@/contentosdb")
	if err != nil {
		fmt.Println(err)
	}
	var lib uint64
	//_ = db.QueryRow("select lib from tokenlibinfo limit 1").Scan(&lib)
	lib = 0
	markedTokens := make(map[string]bool)
	rows, _ := db.Query("select symbol, owner from markedtoken")
	for rows.Next() {
		var symbol string
		var owner string
		if err := rows.Scan(&symbol, &owner); err != nil {
			continue
		}
		key := fmt.Sprintf("%s#%s", symbol, owner)
		markedTokens[key] = true
	}
	rows, _ = db.Query("SELECT block_height, block_log from statelog where block_height > ? limit 1000", lib)
	for rows.Next() {
		var blockHeight uint64
		var log interface{}
		var blockLog iservices.BlockLog
		if err := rows.Scan(&blockHeight, &log); err != nil {
			continue
		}
		data := log.([]byte)
		if err := json.Unmarshal(data, &blockLog); err != nil {
			continue
		}
		//blockId := blockLog.BlockId
		trxLogs := blockLog.TrxLogs
		for _, trxLog := range trxLogs {
			//trxId := trxLog.TrxId
			opLogs := trxLog.OpLogs
			for _, opLog := range opLogs {
				//action := opLog.Action
				property := opLog.Property
				target := opLog.Target
				result := opLog.Result
				if target == "stats" {
					continue
				}
				switch property {
				case "contract":
					mapData := result.(map[string]interface{})
					var contractData itype.ContractData
					for k, v := range mapData{
						err := SetField(&contractData, k, v)
						if err != nil {
							fmt.Println(err)
						}
					}
					data := []byte(contractData.Record)
					var tokenData itype.TokenData
					if err := json.Unmarshal(data, &tokenData); err != nil {
						fmt.Println(err)
					}
					fmt.Println(tokenData)
				}
			}
		}
	}
}
