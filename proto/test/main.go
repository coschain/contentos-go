package main

import (
	"fmt"
	"os"

	"github.com/gogo/protobuf/proto"

	"github.com/coschain/contentos-go/proto/type-proto"
	"github.com/coschain/contentos-go/proto/common-interface"
)

func main() {

	// AccountCreateOperation
	acop := &prototype.AccountCreateOperation{
		Fee:            &prototype.Coin{Amount: &prototype.Safe64{Value: 1}},
		Creator:        &prototype.AccountName{Value: "alice"},
		NewAccountName: &prototype.AccountName{Value: "alice"},
		Owner: &prototype.Authority{
			Cf:              prototype.Authority_active,
			WeightThreshold: 1,
			AccountAuths: []*prototype.KvAccountAuth{
				&prototype.KvAccountAuth{
					Key:   &prototype.AccountName{Value: "alice"},
					Value: 3,
				},
			},
			KeyAuths: []*prototype.KvKeyAuth{
				&prototype.KvKeyAuth{
					Key: &prototype.PublicKeyType{
							Data: []byte{0},
					},
					Value: 23,
				},
			},
		},
	}

	// TransferOperation
	top := &prototype.TransferOperation{
		From:   &prototype.AccountName{Value: "alice"},
		To:     &prototype.AccountName{Value: "alice"},
		Amount: &prototype.Coin{Amount: &prototype.Safe64{Value: 100}},
		Memo:   "this is transfer",
	}

	baseArray := []commoninterface.BaseOperation{}
	baseArray = append(baseArray, acop)
	baseArray = append(baseArray, top)

	for _, elem := range baseArray {
		switch x := elem.(type) {
		case *prototype.AccountCreateOperation:
			fmt.Println("$$$AccountCreateOperation$$$")
			fmt.Println(x)
			fmt.Println("$$$----------------------$$$")
		case *prototype.TransferOperation:
			fmt.Println("$$$TransferOperation$$$")
			fmt.Println(x)
			fmt.Println("$$$-----------------$$$")
		default:
			panic("invalid type")
		}
	}

	// now test marshal and unmarshal
	fmt.Println("now test marshal and unmarshal, write AccountCreateOperation to file then read it into a new object")
	data, err := proto.Marshal(acop)
	if err != nil {
		panic(err)
	}

	fp, _ := os.Create("AccountCreateOperation.serialization")
	len, err := fp.Write(data)
	fmt.Printf("wrote %d bytes\n", len)
	fp.Sync()

	readData := make([]byte, len)
	f, err := os.Open("AccountCreateOperation.serialization")
	readLen, err := f.Read(readData)
	if err != nil {
		panic(err)
	}
	fmt.Printf("read %d bytes\n", readLen)

	acop2 := &prototype.AccountCreateOperation{}
	proto.Unmarshal(readData, acop2)

	fmt.Println(acop2)

	// transaction
	trx := &prototype.Transaction{
		RefBlockNum:    1,
		RefBlockPrefix: 2,
	}

	acopTrx := &prototype.Operation_Op1{}
	acopTrx.Op1 = acop

	topTrx := &prototype.Operation_Op2{}
	topTrx.Op2 = top

	op1 := &prototype.Operation{Op: acopTrx}
	op2 := &prototype.Operation{Op: topTrx}
	trx.Operations = append(trx.Operations, op1)
	trx.Operations = append(trx.Operations, op2)

	for _, elem := range trx.Operations {
		switch x := elem.Op.(type) {
		case *prototype.Operation_Op1:
			fmt.Println("Operation_Op1---> ", x)
		case *prototype.Operation_Op2:
			fmt.Println("Operation_Op2---> ", x)
		case nil:
			fmt.Println("not set")
		default:
			fmt.Println("Op has unexpected type")
		}
	}

	trxdata, err := proto.Marshal(trx)
	if err != nil {
		panic(err)
	}

	fp, _ = os.Create("trx.serialization")
	len, err = fp.Write(trxdata)
	fmt.Printf("wrote %d bytes\n", len)
	fp.Sync()

	readData = make([]byte, len)
	f, err = os.Open("trx.serialization")
	readLen, err = f.Read(readData)
	if err != nil {
		panic(err)
	}
	fmt.Printf("read %d bytes\n", readLen)

	trxNew := &prototype.Transaction{}
	proto.Unmarshal(readData, trxNew)
	for _, elem := range trxNew.Operations {
		switch x := elem.Op.(type) {
		case *prototype.Operation_Op1:
			fmt.Println("Operation_Op1---> ", x)
		case *prototype.Operation_Op2:
			fmt.Println("Operation_Op2---> ", x)
		case nil:
			fmt.Println("not set")
		default:
			fmt.Println("Op has unexpected type")
		}
	}
}
