package main

import "fmt"
import "os"

import "github.com/gogo/protobuf/proto"

import "contentos-go/proto/type-proto"
import "contentos-go/proto/common-interface"

func main() {

	// AccountCreateOperation
	acop := &prototype.AccountCreateOperation{
		Fee:            &prototype.Asset{Amount: &prototype.Safe64{Value: 1}, Symbol: 2},
		Creator:        &prototype.Namex{Value: &prototype.Uint128{Hi: 11, Lo: 12}},
		NewAccountName: &prototype.Namex{Value: &prototype.Uint128{Hi: 11, Lo: 12}},
		Owner: &prototype.Authority{
			Cf:              prototype.Authority_active,
			WeightThreshold: 1,
			AccountAuths: []*prototype.KvAccountAuth{
				&prototype.KvAccountAuth{
					Key:   &prototype.Namex{Value: &prototype.Uint128{Hi: 111, Lo: 112}},
					Value: 3,
				},
			},
			KeyAuths: []*prototype.KvKeyAuth{
				&prototype.KvKeyAuth{
					Key: &prototype.PublicKeyType{
						KeyData: &prototype.PublicKeyData{
							Elems_: []byte{0},
						},
					},
					Value: 23,
				},
			},
		},
	}

	// TransferOperation
	top := &prototype.TransferOperation{
		From:   &prototype.Namex{Value: &prototype.Uint128{Hi: 11, Lo: 12}},
		To:     &prototype.Namex{Value: &prototype.Uint128{Hi: 11, Lo: 12}},
		Amount: &prototype.Asset{Amount: &prototype.Safe64{Value: 100}, Symbol: 2},
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
}
