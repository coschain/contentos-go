package table

import (
	"fmt"
	"github.com/coschain/contentos-go/app/table"
	"github.com/coschain/contentos-go/db/storage"
	"github.com/coschain/contentos-go/prototype"
	"testing"
)

func TestTable(t *testing.T) {
	var (
		tab *Table
		err error
	)
	RegisterProtoType((*prototype.AccountName)(nil))
	RegisterProtoType((*prototype.TimePointSec)(nil))
	RegisterProtoType((*prototype.PublicKeyType)(nil))
	RegisterProtoType((*prototype.Coin)(nil))
	RegisterProtoType((*prototype.Vest)(nil))

	db := storage.NewMemoryDatabase()
	tab, err = ProtoTableBuilder((*table.SoAccount)(nil)).
		Database(db).
		Index("name", Primary).
		Index("created_time", Nonunique).
		Index("pub_key", Unique).
		Index("balance", Nonunique).
		Index("vesting_shares", Nonunique).
		Build()
	if err != nil {
		t.Fatal(err)
	}

	x, err := tab.NewRow(map[string]interface{} {
		"name": &prototype.AccountName{ Value: "alice" },
		"created_time": prototype.NewTimePointSec(1000),
		"creator": &prototype.AccountName{ Value: "initminer" },
		"pub_key": prototype.PublicKeyFromBytes([]byte("public_key")),
		"balance": prototype.NewCoin(1000),
		"vesting_shares": prototype.NewVest(500),
	}).Col().Get()

	fmt.Println(x, err)
}
