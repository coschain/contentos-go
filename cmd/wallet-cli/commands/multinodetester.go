package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"strconv"
	"time"
)

var MultinodetesterCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "multinodetester",
		Short:   "multinodetester count",
		Example: "multinodetester count",
		Args:    cobra.ExactArgs(1),
		Run:     multinodetester,
	}
	return cmd
}

func makeMultiNodeTeseterTrx(count int64, onlyCreate bool) (*prototype.SignedTransaction, error) {

	priKeys := make([]*prototype.PrivateKeyType, 0)

	tx := &prototype.Transaction{RefBlockNum: 0, RefBlockPrefix: 0, Expiration: &prototype.TimePointSec{UtcSeconds: uint32(time.Now().Unix()) + 30}}
	trx := &prototype.SignedTransaction{Trx: tx}

	creator := prototype.NewAccountName(constants.COSInitMiner)

	creatorPriKey, err := prototype.PrivateKeyFromWIF(constants.InitminerPrivKey)
	if err != nil {
		return nil, err
	}

	opCreatorBpVote := &prototype.BpVoteOperation{Voter: creator, Witness: creator, Cancel: false}

	if !onlyCreate {
		trx.Trx.AddOperation(opCreatorBpVote)
	}

	for index := int64(1); index < count; index++ {
		bpName := fmt.Sprintf("%s%d", constants.COSInitMiner, index)
		keys, err := prototype.GenerateNewKeyFromBytes([]byte(bpName))
		if err != nil {
			return nil, err
		}

		pubKey, err := keys.PubKey()
		if err != nil {
			return nil, err
		}

		opCreate := &prototype.AccountCreateOperation{
			Fee:            prototype.NewCoin(1),
			Creator:        creator,
			NewAccountName: &prototype.AccountName{Value: bpName},
			Owner:          prototype.NewAuthorityFromPubKey(pubKey),
		}

		opBpReg := &prototype.BpRegisterOperation{
			Owner:           &prototype.AccountName{Value: bpName},
			Url:             bpName,
			Desc:            bpName,
			BlockSigningKey: pubKey,
			Props: &prototype.ChainProperties{
				AccountCreationFee: prototype.NewCoin(1),
				MaximumBlockSize:   10 * 1024 * 1024,
			},
		}

		opBpVote := &prototype.BpVoteOperation{Voter: prototype.NewAccountName(constants.COSInitMiner), Witness: prototype.NewAccountName(bpName), Cancel: false}

		if !onlyCreate {
			trx.Trx.AddOperation(opBpReg)
			trx.Trx.AddOperation(opBpVote)
			priKeys = append(priKeys, keys)
		} else {
			trx.Trx.AddOperation(opCreate)
		}

	}

	priKeys = append(priKeys, creatorPriKey)

	for _, k := range priKeys {
		res := trx.Sign(k, prototype.ChainId{Value: 0})
		trx.Signatures = append(trx.Signatures, &prototype.SignatureType{Sig: res})
	}

	if err := trx.Validate(); err != nil {
		return nil, err
	}
	return trx, nil
}

func multinodetester(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	idx, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}

	{
		signTx, err := makeMultiNodeTeseterTrx(idx, true)
		if err != nil {
			fmt.Println(err)
			return
		}
		req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
		resp, err := client.BroadcastTrx(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(fmt.Sprintf("create Result: %v", resp))
		}
	}

	{
		signTx, err := makeMultiNodeTeseterTrx(idx, false)
		if err != nil {
			fmt.Println(err)
			return
		}
		req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
		resp, err := client.BroadcastTrx(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(fmt.Sprintf("bpvote Result: %v", resp))
		}
	}

}
