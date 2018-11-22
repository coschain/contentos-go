package commands

import (
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils/mock"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet/mock"
	"github.com/coschain/contentos-go/rpc/mock_grpcpb"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTransferVesting(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)
	mywallet := mock_wallet.NewMockWallet(ctrl)
	myassert := assert.New(t)
	passwordReader := mock_utils.NewMockPasswordReader(ctrl)
	cmd := TransferVestingCmd()
	cmd.SetContext("wallet", mywallet)
	cmd.SetContext("rpcclient", client)
	cmd.SetContext("preader", passwordReader)
	for _, child := range cmd.Commands() {
		child.Context = cmd.Context
	}
	cmd.SetArgs([]string{"initminer", "initminer", "500"})
	priv_account := &wallet.PrivAccount{
		Account: wallet.Account{
			Name:   "initminer",
			PubKey: "COS6oKUcS7jNfPk48SEHENfeHbkWWjH7QAJt6C5tzGyL46yTWWBBv",
		},
		PrivKey: "27Pah3aJ8XbaQxgU1jxmYdUzWaBbBbbxLbZ9whSH9Zc8GbPMhw",
	}
	mywallet.EXPECT().GetUnlockedAccount("initminer").Return(priv_account, true)
	resp := &grpcpb.BroadcastTrxResponse{Status: 1, Msg: "success"}
	client.EXPECT().BroadcastTrx(gomock.Any(), gomock.Any()).Return(resp, nil).Do(func(context interface{}, req *grpcpb.BroadcastTrxRequest) {
		op := req.Transaction.Trx.Operations[0]
		transfer_vesting_op := op.GetOp10()
		myassert.Equal(transfer_vesting_op.From.Value, "initminer")
		myassert.Equal(transfer_vesting_op.To.Value, "initminer")
		myassert.Equal(transfer_vesting_op.Amount.Value, uint64(500))

	})
	_, err := cmd.ExecuteC()
	if err != nil {
		t.Error(err)
	}
}
