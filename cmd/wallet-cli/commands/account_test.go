package commands

import (
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet/mock"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/mock_grpcpb"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)
	wallet := mock_wallet.NewMockWallet(ctrl)
	accCmd := AccountCmd()
	accCmd.SetContext("wallet", wallet)
	accCmd.SetContext("rpcclient", client)
	for _, child := range accCmd.Commands() {
		child.Context = accCmd.Context
	}
	accCmd.SetArgs([]string{"get", "initminer"})
	req := &grpcpb.GetAccountByNameRequest{AccountName: &prototype.AccountName{Value: "initminer"}}
	//resp := &grpcpb.AccountResponse{AccountName: &prototype.AccountName{Value: "initminer"}}
	resp := &grpcpb.AccountResponse{Info: &grpcpb.AccountInfo{AccountName: &prototype.AccountName{Value: "initminer"}}, State: &grpcpb.ChainState{}}
	client.EXPECT().GetAccountByName(gomock.Any(), req).Return(resp, nil)
	_, err := accCmd.ExecuteC()
	assert.NoError(t, err, accCmd)
}

func TestGetNilAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)
	wallet := mock_wallet.NewMockWallet(ctrl)
	accCmd := AccountCmd()
	accCmd.SetContext("wallet", wallet)
	accCmd.SetContext("rpcclient", client)
	for _, child := range accCmd.Commands() {
		child.Context = accCmd.Context
	}
	accCmd.SetArgs([]string{"get", "initminer"})
	req := &grpcpb.GetAccountByNameRequest{AccountName: &prototype.AccountName{Value: "initminer"}}
	resp := &grpcpb.AccountResponse{}
	client.EXPECT().GetAccountByName(gomock.Any(), req).Return(resp, nil)
	_, err := accCmd.ExecuteC()
	assert.NoError(t, err, accCmd)
}
