package commands

import (
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils/mock"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet/mock"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/mock_grpcpb"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/golang/mock/gomock"
	"testing"
)

func TestImportAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)
	mywallet := mock_wallet.NewMockWallet(ctrl)
	passwordReader := mock_utils.NewMockPasswordReader(ctrl)
	cmd := ImportCmd()
	cmd.SetContext("wallet", mywallet)
	cmd.SetContext("rpcclient", client)
	cmd.SetContext("preader", passwordReader)
	for _, child := range cmd.Commands() {
		child.Context = cmd.Context
	}
	cmd.SetArgs([]string{"initminer", "2syFyhZ4kfoS8Sz933nPA3jEUEHPFCsiAB2LUH5HqVjTKJwWGn", "-f"})
	passwordReader.EXPECT().ReadPassword(gomock.Any()).Return([]byte("123456"), nil)
	mywallet.EXPECT().Create("initminer", gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	pubKey, err := prototype.PublicKeyFromWIF("COS8V8KUkBcxUQGkUNByoYLUSvc9ge7kgrGc8wbD7WDX3KXhCnZLz")
	if err != nil {
		t.Error(err)
	}
	resp := &grpcpb.AccountResponse{AccountName: &prototype.AccountName{Value: "initminer"},
		PublicKeys: []*prototype.PublicKeyType{pubKey}}
	client.EXPECT().GetAccountByName(gomock.Any(), gomock.Any()).Return(resp, nil)
	_, err = cmd.ExecuteC()
	if err != nil {
		t.Error(err)
	}
}
