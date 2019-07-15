package utils

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"github.com/kataras/go-errors"
	"hash/crc32"
	"math/rand"
	"regexp"
	"strconv"
	"syscall"
	"time"
)

const MinimumCos = "0.000001"
var EstimateStamina bool

func GenerateSignedTxAndValidate(cmd *cobra.Command, ops []interface{}, signers *wallet.PrivAccount) (*prototype.SignedTransaction, error) {
	client := cmd.Context["rpcclient"].(grpcpb.ApiServiceClient)
	chainId := cmd.Context["chain_id"].(prototype.ChainId)
	return GenerateSignedTxAndValidate2(client, ops, signers, chainId)
}

func GenerateSignedTxAndValidate2(client grpcpb.ApiServiceClient, ops []interface{}, signers *wallet.PrivAccount, chainId prototype.ChainId) (*prototype.SignedTransaction, error) {
	privKey := &prototype.PrivateKeyType{}
	pk, err := prototype.PrivateKeyFromWIF(signers.PrivKey)
	if err != nil {
		return nil, err
	}
	privKey = pk
	return GenerateSignedTxAndValidate3(client, ops, privKey, chainId)
}

func GenerateSignedTxAndValidate3(client grpcpb.ApiServiceClient, ops []interface{}, privKey *prototype.PrivateKeyType, chainId prototype.ChainId) (*prototype.SignedTransaction, error) {
	chainState, err := GetChainState(client)
	if err != nil {
		return nil, err
	}
	return GenerateSignedTxAndValidate4(chainState.Dgpo, 30, ops, privKey, chainId)
}

func GetChainState(client grpcpb.ApiServiceClient) (*grpcpb.ChainState, error) {
	req := &grpcpb.NonParamsRequest{}
	resp, err := client.GetChainState(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp.State, nil
}

func GenerateSignedTxAndValidate4(dgp *prototype.DynamicProperties, expiration uint32, ops []interface{}, privKey *prototype.PrivateKeyType, chainId prototype.ChainId) (*prototype.SignedTransaction, error) {
	refBlockPrefix := common.TaposRefBlockPrefix(dgp.HeadBlockId.Hash)
	// occupant implement
	refBlockNum := common.TaposRefBlockNum(dgp.HeadBlockNumber)
	tx := &prototype.Transaction{RefBlockNum: refBlockNum, RefBlockPrefix: refBlockPrefix, Expiration: &prototype.TimePointSec{UtcSeconds: dgp.Time.UtcSeconds + expiration}}
	for _, op := range ops {
		tx.AddOperation(op)
	}

	signTx := prototype.SignedTransaction{Trx: tx}

	res := signTx.Sign(privKey, chainId)
	signTx.Signature = &prototype.SignatureType{Sig: res}

	if err := signTx.Validate(); err != nil {
		return nil, err
	}

	return &signTx, nil
}

func GenerateUUID(content string) uint64 {
	crc32q := crc32.MakeTable(0xD5828281)
	randContent := content + string(rand.Intn(1e5))
	return uint64(time.Now().Unix()*1e9) + uint64(crc32.Checksum([]byte(randContent), crc32q))
}

func GetPassphrase(reader PasswordReader) (string, error) {
	fmt.Print("Enter passphrase > ")
	bytePassphrase, err := reader.ReadPassword(int(syscall.Stdin))
	fmt.Println()
	if err != nil {
		return "", err
	}
	passphrase := string(bytePassphrase)
	return passphrase, nil
}

func ParseCos(v string) (uint64,error) {
	if m, _ := regexp.MatchString("^[0-9]*\\.[0-9]{6}$", v); !m {
		return 0,errors.New("input must be x.xxxxxx : any bit before . and six bit after .")
	}

	amountFloat, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0,err
	}
	if amountFloat < 0.000001 {
		err = errors.New(fmt.Sprintf("minimum can not less 0.000001"))
		return 0,err
	}
	amount := amountFloat * constants.COSTokenDecimals
	return uint64(amount),nil
}

func ProcessEstimate(cmd *cobra.Command) bool {
	cmd.Flags().BoolVarP(&EstimateStamina,"estimate","",false,"--estimate=true")
	cmd.Flags().Lookup("estimate").NoOptDefVal = "true"
	return EstimateStamina
}

