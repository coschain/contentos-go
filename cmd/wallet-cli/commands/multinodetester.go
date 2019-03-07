package commands

import (
	"context"
	"encoding/binary"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/common/constants"
	//"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"strconv"
	//"time"
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


func makeBpRegVoteTrx(client grpcpb.ApiServiceClient, count int64) (*prototype.SignedTransaction, error) {

	resp, _ := client.GetStatInfo( context.Background(), &grpcpb.NonParamsRequest{} )
	refBlockPrefix := binary.BigEndian.Uint32(resp.State.Dgpo.HeadBlockId.Hash[8:12])
	refBlockNum := uint32(resp.State.Dgpo.HeadBlockNumber & 0x7ff)
	tx := &prototype.Transaction{RefBlockNum: refBlockNum, RefBlockPrefix: refBlockPrefix, Expiration: &prototype.TimePointSec{UtcSeconds: resp.State.Dgpo.Time.UtcSeconds + 30}}
	trx := &prototype.SignedTransaction{Trx: tx}

	bpName := fmt.Sprintf("%s%d", constants.COSInitMiner, count)
	keys, err := prototype.GenerateNewKeyFromBytes([]byte(bpName))
	if err != nil {
		return nil, err
	}

	pubKey, err := keys.PubKey()
	if err != nil {
		return nil, err
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

	opBpVote := &prototype.BpVoteOperation{Voter: prototype.NewAccountName(bpName), Witness: prototype.NewAccountName(bpName), Cancel: false}

	trx.Trx.AddOperation(opBpReg)
	trx.Trx.AddOperation(opBpVote)


	res := trx.Sign(keys, prototype.ChainId{Value: 0})
	trx.Signature = &prototype.SignatureType{Sig: res}

	if err := trx.Validate(); err != nil {
		return nil, err
	}
	return trx, nil
}


func createMNTAccountTrx(client grpcpb.ApiServiceClient, count int64) (*prototype.SignedTransaction, error) {

	resp, _ := client.GetStatInfo( context.Background(), &grpcpb.NonParamsRequest{} )
	refBlockPrefix := binary.BigEndian.Uint32(resp.State.Dgpo.HeadBlockId.Hash[8:12])
	refBlockNum := uint32(resp.State.Dgpo.HeadBlockNumber & 0x7ff)
	tx := &prototype.Transaction{RefBlockNum: refBlockNum, RefBlockPrefix: refBlockPrefix, Expiration: &prototype.TimePointSec{UtcSeconds: resp.State.Dgpo.Time.UtcSeconds + 30}}
	trx := &prototype.SignedTransaction{Trx: tx}

	creator := prototype.NewAccountName(constants.COSInitMiner)

	creatorPriKey, err := prototype.PrivateKeyFromWIF(constants.InitminerPrivKey)
	if err != nil {
		return nil, err
	}

	opCreatorBpVote := &prototype.BpVoteOperation{Voter: creator, Witness: creator, Cancel: false}

	trx.Trx.AddOperation(opCreatorBpVote)

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

		trx.Trx.AddOperation(opCreate)
	}

	res := trx.Sign(creatorPriKey, prototype.ChainId{Value: 0})
	trx.Signature = &prototype.SignatureType{Sig: res}

	if err := trx.Validate(); err != nil {
		return nil, err
	}
	return trx, nil
}

/*
func makeMultiNodeTeseterTrx(count int64, onlyCreate bool) (*prototype.SignedTransaction, error) {

	var priKey *prototype.PrivateKeyType = nil

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

		opBpVote := &prototype.BpVoteOperation{Voter: prototype.NewAccountName(bpName), Witness: prototype.NewAccountName(bpName), Cancel: false}

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
}*/

func multinodetester(cmd *cobra.Command, args []string) {
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	idx, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Println(err)
		return
	}

	{
		signTx, err := createMNTAccountTrx(client, idx)
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

	var i int64
	for i = 1; i < idx; i++ {
		signTx, err := makeBpRegVoteTrx(client, i)
		if err != nil {
			fmt.Println(err)
			return
		}
		req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
		req.OnlyDeliver = true
		resp, err := client.BroadcastTrx(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(fmt.Sprintf("bpvote Result: %v", resp))
		}
	}
}
