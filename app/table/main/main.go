package main

import (
	"fmt"
	"time"
	base "github.com/coschain/contentos-go/proto/type-proto"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/app/table"

)


func main() {
	db := storage.NewMemoryDatabase()
	defer db.Close()

	acc  := &base.AccountName{ Value:"yykingking" }
	wrap := table.NewSoAccountWrap( db, acc)

	fmt.Println( "CheckExist0:" , wrap.CheckExist() )

	newAcc := &table.SoAccount{}

	newAcc.CreatedTime = &base.TimePointSec{ UtcSeconds:0 }
	newAcc.Creator	= &base.AccountName{ Value:"Jack" }
	newAcc.PubKey  = &base.PublicKeyType{ Data:nil }

	fmt.Println( "CreateAccount:" , 	wrap.CreateAccount(newAcc) )


	fmt.Println( "CheckExist1:" , wrap.CheckExist() )

	fmt.Println( "GetAccountCreator:" , wrap.GetAccountCreator() )


	begin:= time.Now().UnixNano()
	for index := 0; index <= 1000000; index++ {
		_ = wrap.GetAccountCreator()
		_ = wrap.GetAccountCreatedTime()

	}
	cost := (time.Now().UnixNano() - begin) / 1000000.0

	fmt.Println("Cost time average: ", cost, " ns")
}
