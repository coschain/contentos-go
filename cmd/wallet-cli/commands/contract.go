package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
)

var ContractCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "contract",
		Short:   "query contract info",
		Example: "contract [owner] [contract_name]",
		Args:    cobra.ExactArgs(2),
		Run:     queryContract,
	}
	return cmd
}

func queryContract(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	rpc := c.(grpcpb.ApiServiceClient)
	owner := args[0]
	contract := args[1]

	req := &grpcpb.GetContractInfoRequest{
		Owner: &prototype.AccountName{Value: owner},
		ContractName:contract,
		FetchAbi:true,
		FetchCode:true,
	}
	resp, err := rpc.GetContractInfo(context.Background(), req)
	if err != nil {
		fmt.Println(err)
	} else {
		buf, _ := json.MarshalIndent(resp, "", "\t")
		fmt.Println(fmt.Sprintf("GetContractInfo detail: %s", buf))
	}
}
