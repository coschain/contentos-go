package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/rpc/pb"
)

var VmTableCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use: "table",
		Short: "query vm table content",
		Example: "table owner_name contract_name table_name field_name field_begin field_end",
		Args:    cobra.MinimumNArgs(6),
		Run: queryTable,
	}

	return cmd
}

func queryTable(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	rpc := c.(grpcpb.ApiServiceClient)
	owner := args[0]
	contract := args[1]
	table := args[2]
	field := args[3]
	begin := args[4]
	end := args[5]

	req := &grpcpb.GetTableContentRequest{
		Owner:owner,
		Contranct:contract,
		Table:table,
		Field:field,
		Begin:begin,
		End:end,
	}
	resp, err := rpc.QueryTableContent(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		buf, _ := json.MarshalIndent(resp, "", "\t")
		fmt.Println(fmt.Sprintf("queryTable detail: %s", buf))
	}
}