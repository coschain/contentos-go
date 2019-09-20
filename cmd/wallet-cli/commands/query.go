package commands

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/coschain/cobra"
	grpcpb "github.com/coschain/contentos-go/rpc/pb"
	"strings"
)

var QueryCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "query <table_name> <json_of_key>",
		Short:   "query any app table record",
		Example: strings.Join([]string{"query Account \\\"initminer\\\"", "query BlockProducerVote {\\\"block_producer\\\":\\\"initminer\\\",\\\"voter\\\":\\\"initminer\\\"}"}, "\n"),
		Args:    cobra.ExactArgs(2),
		Run:     query,
	}
	return cmd
}

func query(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	req := &grpcpb.GetAppTableRecordRequest{
		TableName:            args[0],
		Key:                  args[1],
	}
	if resp, err := client.GetAppTableRecord(context.Background(), req); err != nil {
		fmt.Println("rpc error:", err.Error())
		return
	} else if !resp.GetSuccess() {
		fmt.Println("failed:", resp.GetErrorMsg())
	} else {
		var out bytes.Buffer
		_ = json.Indent(&out, []byte(resp.GetRecord()), "", "  ")
		fmt.Println(string(out.Bytes()))
	}
}
