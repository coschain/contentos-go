package prototype

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/gogo/protobuf/proto"
)

func Test_Serialize(t *testing.T) {

	// AccountCreateOperation
	acop := &AccountCreateOperation{
		Fee:            NewCoin(1),
		Creator:        &AccountName{Value: "alice"},
		NewAccountName: &AccountName{Value: "alice"},
		Owner: &Authority{
			Cf:              Authority_active,
			WeightThreshold: 1,
			AccountAuths: []*KvAccountAuth{
				&KvAccountAuth{
					Name:    &AccountName{Value: "alice"},
					Weight: 3,
				},
			},
			KeyAuths: []*KvKeyAuth{
				&KvKeyAuth{
					Key: &PublicKeyType{
						Data: []byte{0},
					},
					Weight: 23,
				},
			},
		},
	}

	// TransferOperation
	top := &TransferOperation{
		From:   &AccountName{Value: "alice"},
		To:     &AccountName{Value: "alice"},
		Amount: NewCoin(100),
		Memo:   "this is transfer",
	}

	// test AccountCreateOperation marshal and unmarshal
	fmt.Println("=== test marshal and unmarshal, write AccountCreateOperation to file then read it into a new object ===")
	data, err := proto.Marshal(acop)
	if err != nil {
		t.Error("AccountCreateOperation Marshal failed")
	}

	fp, _ := os.Create("AccountCreateOperation.serialization")
	len, err := fp.Write(data)
	//fmt.Printf("wrote %d bytes\n", len)
	fp.Sync()

	readData := make([]byte, len)
	f, err := os.Open("AccountCreateOperation.serialization")
	_, err = f.Read(readData)
	if err != nil {
		t.Error("AccountCreateOperation file read failed")
	}
	//fmt.Printf("read %d bytes\n", readLen)

	acop2 := &AccountCreateOperation{}
	err = proto.Unmarshal(readData, acop2)
	if err != nil {
		t.Error("AccountCreateOperation Unmarshal failed")
	}

	//fmt.Println(acop2)

	// transaction
	trx := &Transaction{
		RefBlockNum:    1,
		RefBlockPrefix: 2,
	}

	acopTrx := &Operation_Op1{}
	acopTrx.Op1 = acop

	topTrx := &Operation_Op2{}
	topTrx.Op2 = top

	op1 := &Operation{Op: acopTrx}
	op2 := &Operation{Op: topTrx}
	trx.Operations = append(trx.Operations, op1)
	trx.Operations = append(trx.Operations, op2)

	/*
		for _, elem := range trx.Operations {
			switch x := elem.Op.(type) {
			case *Operation_Op1:
				fmt.Println("Operation_Op1---> ", x)
			case *Operation_Op2:
				fmt.Println("Operation_Op2---> ", x)
			case nil:
				fmt.Println("not set")
			default:
				t.Error("Op has unexpected type")
			}
		}*/

	trxdata, err := proto.Marshal(trx)
	if err != nil {
		t.Error("trx Marshal failed")
	}

	fp, _ = os.Create("trx.serialization")
	len, err = fp.Write(trxdata)
	//fmt.Printf("wrote %d bytes\n", len)
	fp.Sync()

	readData = make([]byte, len)
	f, err = os.Open("trx.serialization")
	_, err = f.Read(readData)
	if err != nil {
		t.Error("trx file read failed")
	}
	//fmt.Printf("read %d bytes\n", readLen)

	trxNew := &Transaction{}
	err = proto.Unmarshal(readData, trxNew)
	if err != nil {
		t.Error("trx file read failed")
	}
	for _, elem := range trxNew.Operations {
		switch x := elem.Op.(type) {
		case *Operation_Op1:
			if !compare(*x.Op1, *acop) {
				t.Error("trx Marshal and Unmarshal Error, op1")
			}
			//fmt.Println("Operation_Op1---> ", x)
		case *Operation_Op2:
			if !compare(*x.Op2, *top) {
				t.Error("trx Marshal and Unmarshal Error, op2")
			}
		//	fmt.Println("Operation_Op2---> ", x)
		case nil:
			fmt.Println("not set")
		default:
			fmt.Println("Op has unexpected type")
		}
	}
	fmt.Println("=== test marshal and unmarshal pass ===")
}

func compare(a interface{}, b interface{}) bool {
	ta := reflect.TypeOf(a)
	tb := reflect.TypeOf(b)
	fmt.Println("typeA:", ta.String(), " typeB:", tb.String())
	if ta.String() != tb.String() {
		return false
	}

	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)

	// slice
	if va.Kind() == reflect.Slice ||
		va.Kind() == reflect.Array {
		if va.Len() != vb.Len() {
			return false
		}
		if va.Len() == 0 {
			return true
		}
		for i := 0; i < va.Len(); i++ {
			if !isBasicType(va.Index(i).Kind()) {
				if va.Index(i).Kind() == reflect.Ptr {
					struct_va := reflect.Indirect(va.Index(i))
					struct_vb := reflect.Indirect(vb.Index(i))
					if !compare(struct_va.Interface(), struct_vb.Interface()) {
						return false
					}
				} else {
					if !compare(va.Index(i).Interface(), vb.Index(i).Interface()) {
						return false
					}
				}
			} else {
				valueA := va.Index(i).Interface()
				valueB := vb.Index(i).Interface()
				fmt.Println(valueA, " <-value in array-> ", valueB)
				if valueA != valueB {
					return false
				}
			}
		}
		return true
	} else if va.Kind() == reflect.Map {
	} else {
	}

	fmt.Println("numField:", ta.NumField())
	if ta.NumField() != tb.NumField() {
		return false
	}
	if ta.NumField() == 0 {
		return true
	}

	for i := 0; i < ta.NumField(); i++ {

		typeA := ta.Field(i)
		typeB := tb.Field(i)

		// filter nil , null
		if va.Field(i).Kind() == reflect.Ptr {
			if va.Field(i).IsNil() && va.Field(i).IsNil() {
				continue
			} else if !va.Field(i).IsNil() && !va.Field(i).IsNil() {

			} else {
				return false
			}
		}
		//

		//fmt.Println("FieldTypeA:", typeA.Name, " FieldTypeB:", typeB.Name)
		if typeA.Name != typeB.Name {
			return false
		}

		// filter pb generate struct
		if strings.Contains(typeA.Name, "XXX_") {
			//	fmt.Println("filter pb field")
			continue
		}
		//

		if !isBasicType(va.Field(i).Kind()) {
			if va.Field(i).Kind() == reflect.Ptr {
				struct_va := reflect.Indirect(va.Field(i))
				struct_vb := reflect.Indirect(vb.Field(i))
				if !compare(struct_va.Interface(), struct_vb.Interface()) {
					return false
				}
			} else {
				if !compare(va.Field(i).Interface(), vb.Field(i).Interface()) {
					return false
				}
			}
		} else {
			valueA := va.Field(i).Interface()
			valueB := vb.Field(i).Interface()
			fmt.Println(valueA, " <-value-> ", valueB)
			if valueA != valueB {
				return false
			}
		}
	}

	return true
}

func isBasicType(k reflect.Kind) bool {

	return k == reflect.Bool ||
		k == reflect.Int ||
		k == reflect.Int8 ||
		k == reflect.Int16 ||
		k == reflect.Int32 ||
		k == reflect.Int64 ||
		k == reflect.Uint ||
		k == reflect.Uint8 ||
		k == reflect.Uint16 ||
		k == reflect.Uint32 ||
		k == reflect.Uint64 ||
		k == reflect.Float32 ||
		k == reflect.Float64 ||
		k == reflect.String ||
		k == reflect.Complex64 ||
		k == reflect.Complex128

}
