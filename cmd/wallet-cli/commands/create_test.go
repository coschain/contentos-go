package commands

import (
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils/mock"
	wallet2 "github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet/mock"
	"github.com/coschain/contentos-go/rpc/mock_grpcpb"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCreateAccount(t *testing.T) {
	ctrl := gomock.NewController(t)
	client := mock_grpcpb.NewMockApiServiceClient(ctrl)
	wallet := mock_wallet.NewMockWallet(ctrl)
	passwordReader := mock_utils.NewMockPasswordReader(ctrl)
	cmd := CreateCmd()
	cmd.SetContext("wallet", wallet)
	cmd.SetContext("rpcclient", client)
	cmd.SetContext("preader", passwordReader)
	for _, child := range cmd.Commands() {
		child.Context = cmd.Context
	}
	priv_account := &wallet2.PrivAccount{
		Account: wallet2.Account{
			Name:   "initminer",
			PubKey: "COS5xKso6RQz62BtrfwsRVZ9XXjiqJN7kqjjrwcFCXt3amc1AQLuU",
		},
		PrivKey: "2i3yqxhyw9z56CXUp5xmHBe9LcDrj2UeemQuWt4jUQCCCNaauo",
	}
	wallet.EXPECT().GenerateNewKey().Return(
		"COS8V8KUkBcxUQGkUNByoYLUSvc9ge7kgrGc8wbD7WDX3KXhCnZLz",
		"2syFyhZ4kfoS8Sz933nPA3jEUEHPFCsiAB2LUH5HqVjTKJwWGn", nil)
	wallet.EXPECT().GetUnlockedAccount("initminer").Return(priv_account, true)
	wallet.EXPECT().Create("alice", gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	passwordReader.EXPECT().ReadPassword(gomock.Any()).Return([]byte("123456"), nil)
	cmd.SetArgs([]string{"initminer", "alice"})
	resp := &grpcpb.BroadcastTrxResponse{Status: 1, Msg: "success"}
	client.EXPECT().BroadcastTrx(gomock.Any(), gomock.Any()).Return(resp, nil)
	_, err := cmd.ExecuteC()
	assert.NoError(t, err, cmd)
}
