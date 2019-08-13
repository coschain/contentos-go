package commands

import (
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils/mock"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet/mock"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/mock_grpcpb"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestBpRegisterWithoutFlags(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)
	mywallet := mock_wallet.NewMockWallet(ctrl)
	passwordReader := mock_utils.NewMockPasswordReader(ctrl)
	cmd := BpCmd()
	cmd.SetContext("wallet", mywallet)
	cmd.SetContext("rpcclient", client)
	cmd.SetContext("preader", passwordReader)
	cmd.SetContext("chain_id", prototype.ChainId{})
	for _, child := range cmd.Commands() {
		child.Context = cmd.Context
	}
	cmd.SetArgs([]string{"register", "initminer", "COS5JVLLcTPhq4Unr194JzWPDNSYGoMcam8yxnsjgRVo3Nb7ioyFW"})
	priv_account := &wallet.PrivAccount{
		Account: wallet.Account{
			Name:   "initminer",
			PubKey: "COS5JVLLcTPhq4Unr194JzWPDNSYGoMcam8yxnsjgRVo3Nb7ioyFW",
		},
		PrivKey: "4DjYx2KAGh1NP3dai7MZTLUBMMhMBPmwouKE8jhVSESywccpVZ",
	}
	mywallet.EXPECT().GetUnlockedAccount("initminer").Return(priv_account, true)
	mock_utils.NeedChainState(client)
	resp := &grpcpb.BroadcastTrxResponse{Status: 1, Msg: "success"}
	client.EXPECT().BroadcastTrx(gomock.Any(), gomock.Any()).Return(resp, nil)
	_, err := cmd.ExecuteC()
	if err != nil {
		t.Error(err)
	}
}

func TestBpRegisterWithUrl(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)
	mywallet := mock_wallet.NewMockWallet(ctrl)
	myassert := assert.New(t)
	passwordReader := mock_utils.NewMockPasswordReader(ctrl)
	cmd := BpCmd()
	cmd.SetContext("wallet", mywallet)
	cmd.SetContext("rpcclient", client)
	cmd.SetContext("preader", passwordReader)
	cmd.SetContext("chain_id", prototype.ChainId{})
	for _, child := range cmd.Commands() {
		child.Context = cmd.Context
	}
	cmd.SetArgs([]string{"register", "initminer", "COS5JVLLcTPhq4Unr194JzWPDNSYGoMcam8yxnsjgRVo3Nb7ioyFW", "--url", "http://example.com"})
	priv_account := &wallet.PrivAccount{
		Account: wallet.Account{
			Name:   "initminer",
			PubKey: "COS5JVLLcTPhq4Unr194JzWPDNSYGoMcam8yxnsjgRVo3Nb7ioyFW",
		},
		PrivKey: "4DjYx2KAGh1NP3dai7MZTLUBMMhMBPmwouKE8jhVSESywccpVZ",
	}
	mywallet.EXPECT().GetUnlockedAccount("initminer").Return(priv_account, true)
	mock_utils.NeedChainState(client)
	resp := &grpcpb.BroadcastTrxResponse{Status: 1, Msg: "success"}
	client.EXPECT().BroadcastTrx(gomock.Any(), gomock.Any()).Return(resp, nil).Do(func(context interface{}, req *grpcpb.BroadcastTrxRequest) {
		op := req.Transaction.Trx.Operations[0]
		bp_op := op.GetOp3()
		myassert.Equal(bp_op.Owner.Value, "initminer")
		myassert.Equal(bp_op.Url, "http://example.com")
	})
	_, err := cmd.ExecuteC()
	if err != nil {
		t.Error(err)
	}
}

func TestBpRegisterWithDesc(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)
	mywallet := mock_wallet.NewMockWallet(ctrl)
	myassert := assert.New(t)
	passwordReader := mock_utils.NewMockPasswordReader(ctrl)
	cmd := BpCmd()
	cmd.SetContext("wallet", mywallet)
	cmd.SetContext("rpcclient", client)
	cmd.SetContext("preader", passwordReader)
	cmd.SetContext("chain_id", prototype.ChainId{})
	for _, child := range cmd.Commands() {
		child.Context = cmd.Context
	}
	cmd.SetArgs([]string{"register", "initminer", "COS5JVLLcTPhq4Unr194JzWPDNSYGoMcam8yxnsjgRVo3Nb7ioyFW", "--desc", "hello world"})
	priv_account := &wallet.PrivAccount{
		Account: wallet.Account{
			Name:   "initminer",
			PubKey: "COS5JVLLcTPhq4Unr194JzWPDNSYGoMcam8yxnsjgRVo3Nb7ioyFW",
		},
		PrivKey: "4DjYx2KAGh1NP3dai7MZTLUBMMhMBPmwouKE8jhVSESywccpVZ",
	}
	mywallet.EXPECT().GetUnlockedAccount("initminer").Return(priv_account, true)
	mock_utils.NeedChainState(client)
	resp := &grpcpb.BroadcastTrxResponse{Status: 1, Msg: "success"}
	client.EXPECT().BroadcastTrx(gomock.Any(), gomock.Any()).Return(resp, nil).Do(func(context interface{}, req *grpcpb.BroadcastTrxRequest) {
		op := req.Transaction.Trx.Operations[0]
		bp_op := op.GetOp3()
		myassert.Equal(bp_op.Owner.Value, "initminer")
		myassert.Equal(bp_op.Desc, "hello world")
	})
	_, err := cmd.ExecuteC()
	if err != nil {
		t.Error(err)
	}
}

func TestBpRegisterWithFee(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)
	mywallet := mock_wallet.NewMockWallet(ctrl)
	myassert := assert.New(t)
	passwordReader := mock_utils.NewMockPasswordReader(ctrl)
	cmd := BpCmd()
	cmd.SetContext("wallet", mywallet)
	cmd.SetContext("rpcclient", client)
	cmd.SetContext("preader", passwordReader)
	cmd.SetContext("chain_id", prototype.ChainId{})
	for _, child := range cmd.Commands() {
		child.Context = cmd.Context
	}
	cmd.SetArgs([]string{"register", "initminer", "COS5JVLLcTPhq4Unr194JzWPDNSYGoMcam8yxnsjgRVo3Nb7ioyFW", "--fee", "100"})
	priv_account := &wallet.PrivAccount{
		Account: wallet.Account{
			Name:   "initminer",
			PubKey: "COS5JVLLcTPhq4Unr194JzWPDNSYGoMcam8yxnsjgRVo3Nb7ioyFW",
		},
		PrivKey: "4DjYx2KAGh1NP3dai7MZTLUBMMhMBPmwouKE8jhVSESywccpVZ",
	}
	mywallet.EXPECT().GetUnlockedAccount("initminer").Return(priv_account, true)
	mock_utils.NeedChainState(client)
	resp := &grpcpb.BroadcastTrxResponse{Status: 1, Msg: "success"}
	client.EXPECT().BroadcastTrx(gomock.Any(), gomock.Any()).Return(resp, nil).Do(func(context interface{}, req *grpcpb.BroadcastTrxRequest) {
		op := req.Transaction.Trx.Operations[0]
		bp_op := op.GetOp3()
		myassert.Equal(bp_op.Owner.Value, "initminer")
		myassert.Equal(bp_op.Props.AccountCreationFee.Value, uint64(100))
	})
	_, err := cmd.ExecuteC()
	if err != nil {
		t.Error(err)
	}
}

func TestBpRegisterWithBlockSize(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)
	mywallet := mock_wallet.NewMockWallet(ctrl)
	myassert := assert.New(t)
	passwordReader := mock_utils.NewMockPasswordReader(ctrl)
	cmd := BpCmd()
	cmd.SetContext("wallet", mywallet)
	cmd.SetContext("rpcclient", client)
	cmd.SetContext("preader", passwordReader)
	cmd.SetContext("chain_id", prototype.ChainId{})
	for _, child := range cmd.Commands() {
		child.Context = cmd.Context
	}
	cmd.SetArgs([]string{"register", "initminer", "COS5JVLLcTPhq4Unr194JzWPDNSYGoMcam8yxnsjgRVo3Nb7ioyFW"})
	priv_account := &wallet.PrivAccount{
		Account: wallet.Account{
			Name:   "initminer",
			PubKey: "COS5JVLLcTPhq4Unr194JzWPDNSYGoMcam8yxnsjgRVo3Nb7ioyFW",
		},
		PrivKey: "4DjYx2KAGh1NP3dai7MZTLUBMMhMBPmwouKE8jhVSESywccpVZ",
	}
	mywallet.EXPECT().GetUnlockedAccount("initminer").Return(priv_account, true)
	mock_utils.NeedChainState(client)
	resp := &grpcpb.BroadcastTrxResponse{Status: 1, Msg: "success"}
	client.EXPECT().BroadcastTrx(gomock.Any(), gomock.Any()).Return(resp, nil).Do(func(context interface{}, req *grpcpb.BroadcastTrxRequest) {
		op := req.Transaction.Trx.Operations[0]
		bp_op := op.GetOp3()
		myassert.Equal(bp_op.Owner.Value, "initminer")
	})
	_, err := cmd.ExecuteC()
	if err != nil {
		t.Error(err)
	}
}

func TestBpVoteWithoutFlags(t *testing.T) {
	ctrl := gomock.NewController(t)
	myassert := assert.New(t)
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)
	mywallet := mock_wallet.NewMockWallet(ctrl)
	passwordReader := mock_utils.NewMockPasswordReader(ctrl)
	cmd := BpCmd()
	cmd.SetContext("wallet", mywallet)
	cmd.SetContext("rpcclient", client)
	cmd.SetContext("preader", passwordReader)
	cmd.SetContext("chain_id", prototype.ChainId{})
	for _, child := range cmd.Commands() {
		child.Context = cmd.Context
	}
	cmd.SetArgs([]string{"vote", "initminer", "initminer"})
	priv_account := &wallet.PrivAccount{
		Account: wallet.Account{
			Name:   "initminer",
			PubKey: "COS5JVLLcTPhq4Unr194JzWPDNSYGoMcam8yxnsjgRVo3Nb7ioyFW",
		},
		PrivKey: "4DjYx2KAGh1NP3dai7MZTLUBMMhMBPmwouKE8jhVSESywccpVZ",
	}
	mywallet.EXPECT().GetUnlockedAccount("initminer").Return(priv_account, true)
	mock_utils.NeedChainState(client)
	resp := &grpcpb.BroadcastTrxResponse{Status: 1, Msg: "success"}
	client.EXPECT().BroadcastTrx(gomock.Any(), gomock.Any()).Return(resp, nil).Do(func(context interface{}, req *grpcpb.BroadcastTrxRequest) {
		op := req.Transaction.Trx.Operations[0]
		bp_op := op.GetOp5()
		myassert.Equal(bp_op.Cancel, false)
	})
	_, err := cmd.ExecuteC()
	if err != nil {
		t.Error(err)
	}
}

func TestBpVoteCancel(t *testing.T) {
	ctrl := gomock.NewController(t)
	myassert := assert.New(t)
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)
	mywallet := mock_wallet.NewMockWallet(ctrl)
	passwordReader := mock_utils.NewMockPasswordReader(ctrl)
	cmd := BpCmd()
	cmd.SetContext("wallet", mywallet)
	cmd.SetContext("rpcclient", client)
	cmd.SetContext("preader", passwordReader)
	cmd.SetContext("chain_id", prototype.ChainId{})
	for _, child := range cmd.Commands() {
		child.Context = cmd.Context
	}
	cmd.SetArgs([]string{"vote", "initminer", "initminer", "-c"})
	priv_account := &wallet.PrivAccount{
		Account: wallet.Account{
			Name:   "initminer",
			PubKey: "COS5JVLLcTPhq4Unr194JzWPDNSYGoMcam8yxnsjgRVo3Nb7ioyFW",
		},
		PrivKey: "4DjYx2KAGh1NP3dai7MZTLUBMMhMBPmwouKE8jhVSESywccpVZ",
	}
	mywallet.EXPECT().GetUnlockedAccount("initminer").Return(priv_account, true)
	mock_utils.NeedChainState(client)
	resp := &grpcpb.BroadcastTrxResponse{Status: 1, Msg: "success"}
	client.EXPECT().BroadcastTrx(gomock.Any(), gomock.Any()).Return(resp, nil).Do(func(context interface{}, req *grpcpb.BroadcastTrxRequest) {
		op := req.Transaction.Trx.Operations[0]
		bp_op := op.GetOp5()
		myassert.Equal(bp_op.Cancel, true)
	})
	_, err := cmd.ExecuteC()
	if err != nil {
		t.Error(err)
	}
}

func TestBpVoteUnsetFlag(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)
	mock_utils.NeedChainState(client)
	mywallet := mock_wallet.NewMockWallet(ctrl)
	myassert := assert.New(t)
	passwordReader := mock_utils.NewMockPasswordReader(ctrl)
	cmd := BpCmd()
	cmd.SetContext("wallet", mywallet)
	cmd.SetContext("rpcclient", client)
	cmd.SetContext("preader", passwordReader)
	cmd.SetContext("chain_id", prototype.ChainId{})
	for _, child := range cmd.Commands() {
		child.Context = cmd.Context
	}
	cmd.SetArgs([]string{"vote", "initminer", "initminer", "-c"})
	priv_account := &wallet.PrivAccount{
		Account: wallet.Account{
			Name:   "initminer",
			PubKey: "COS5JVLLcTPhq4Unr194JzWPDNSYGoMcam8yxnsjgRVo3Nb7ioyFW",
		},
		PrivKey: "4DjYx2KAGh1NP3dai7MZTLUBMMhMBPmwouKE8jhVSESywccpVZ",
	}
	mywallet.EXPECT().GetUnlockedAccount("initminer").Return(priv_account, true).MaxTimes(100)
	resp := &grpcpb.BroadcastTrxResponse{Status: 1, Msg: "success"}
	client.EXPECT().BroadcastTrx(gomock.Any(), gomock.Any()).Return(resp, nil).Do(func(context interface{}, req *grpcpb.BroadcastTrxRequest) {
		op := req.Transaction.Trx.Operations[0]
		bp_op := op.GetOp5()
		myassert.Equal(bp_op.Cancel, true)
	})
	_, err := cmd.ExecuteC()
	if err != nil {
		t.Error(err)
	}
	client.EXPECT().BroadcastTrx(gomock.Any(), gomock.Any()).Return(resp, nil).Do(func(context interface{}, req *grpcpb.BroadcastTrxRequest) {
		op := req.Transaction.Trx.Operations[0]
		bp_op := op.GetOp5()
		myassert.Equal(bp_op.Cancel, false)
	})
	cmd.SetArgs([]string{"vote", "initminer", "initminer"})
	_, err = cmd.ExecuteC()
	if err != nil {
		t.Error(err)
	}
	client.EXPECT().BroadcastTrx(gomock.Any(), gomock.Any()).Return(resp, nil).Do(func(context interface{}, req *grpcpb.BroadcastTrxRequest) {
		op := req.Transaction.Trx.Operations[0]
		bp_op := op.GetOp5()
		myassert.NotEqual(bp_op.Cancel, false)
	})
	cmd.SetArgs([]string{"vote", "initminer", "initminer", "-c"})
	_, err = cmd.ExecuteC()
	if err != nil {
		t.Error(err)
	}
}
