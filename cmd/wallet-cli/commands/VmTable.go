package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/rpc/pb"
	"strconv"
)

var VmTableCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use: "table",
		Short: "query vm table content",
		Example: "table owner_name contract_name table_name field_name field_begin count(max value:100) [reverse]",
		Args:    cobra.MinimumNArgs(3),
		Run: queryTable,
	}

	return cmd
}

func queryTable(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	rpc := c.(grpcpb.ApiServiceClient)
	argsLen := len(args)
	if argsLen != 7 && argsLen != 3 && argsLen != 6 {
		fmt.Println("invalid parameter count, should be 3 or 6 or 7")
		return
	}

	owner := args[0]
	contract := args[1]
	table := args[2]
	field := "id"
	begin := ""
	count := 30
	reverse := false

	if argsLen == 6 || argsLen == 7 {
		field = args[3]
		begin = args[4]

		argv5, err := strconv.ParseInt(args[5], 10, 32)
		if err != nil {
			fmt.Println(err)
			return
		}
		count = int(argv5)

		if argsLen == 7{
			if args[6] == "true" {
				reverse = true
			}
		}
	}

	if count == 0 {
		fmt.Println("query count cant be 0")
		return
	}


	req := &grpcpb.GetTableContentRequest{
		Owner:     owner,
		Contract: contract,
		Table:     table,
		Field:     field,
		Begin:     begin,
		Count:     uint32(count),
		Reverse:	reverse,
	}
	resp, err := rpc.QueryTableContent(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(fmt.Sprintf("queryTable detail: \n\n%s\n\n", resp.TableContent) )
	}

}