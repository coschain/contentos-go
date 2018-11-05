package main

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/db/storage"
	base "github.com/coschain/contentos-go/common/prototype"
)

func main() {
	//db, _ := storage.NewLevelDatabase("/Users/yykingking/abc123.db")
	db := storage.NewMemoryDatabase()

	defer db.Close()

	for index := 0; index < 10; index++ {
		acc := base.MakeAccountName(fmt.Sprintf("TUser%d", index))
		wrap := table.NewSoAccountWrap(db, acc)
		newAcc := &table.SoAccount{}
		newAcc.CreatedTime = base.MakeTimeSecondPoint(uint32(10 + index))
		newAcc.Creator = base.MakeAccountName(fmt.Sprintf("Jack%d", index))
		newAcc.PubKey = base.MakePublicKeyType(nil)
		newAcc.Name = acc

		wrap.CreateAccount(newAcc)
	}

	{
		lwrap := table.SListAccountByCreatedTime{db}
		iter := lwrap.DoList(*base.MakeTimeSecondPoint(10), *base.MakeTimeSecondPoint(14))
		if iter != nil {
			for iter.Next() {
				fmt.Println("iter1:" , lwrap.GetMainVal(iter), " - ",lwrap.GetSubVal(iter))
			}
		}
	}

	// modify

	{
		acc := base.MakeAccountName("TUser2")
		wrap := table.NewSoAccountWrap(db, acc)

		if wrap.CheckExist() {
			oldTime := wrap.GetAccountCreatedTime()
			oldTime.UtcSeconds += 10
			fmt.Println("modify : ", wrap.MdAccountCreatedTime(*oldTime))
		}
	}

	{
		lwrap := table.SListAccountByCreatedTime{db}
		iter := lwrap.DoList(*base.MakeTimeSecondPoint(10), *base.MakeTimeSecondPoint(14))
		if iter != nil {
			for iter.Next() {
				fmt.Println("iter2:" , lwrap.GetMainVal(iter), " - ",lwrap.GetSubVal(iter))
			}
		}
	}

	{
		acc := base.MakeAccountName("TUser3")
		wrap := table.NewSoAccountWrap(db, acc)

		if wrap.CheckExist() {
			fmt.Println("remove : ", wrap.RemoveAccount())
		}
	}
	{
		lwrap := table.SListAccountByCreatedTime{db}
		iter := lwrap.DoList(*base.MakeTimeSecondPoint(10), *base.MakeTimeSecondPoint(14))
		if iter != nil {
			for iter.Next() {
				fmt.Println("iter3:" , lwrap.GetMainVal(iter), " - ",lwrap.GetSubVal(iter))
			}
		}
	}
}
