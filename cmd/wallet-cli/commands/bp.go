package commands

import (
	"context"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/wallet-cli/commands/utils"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
	"github.com/coschain/contentos-go/rpc/pb"
	"math"
)

const (
	bpUpdateInvalidUint64 = math.MaxUint64
	bpUpdateInvalidUint32 = math.MaxUint32
	bpUpdateInvalidString = ""
)

var bpUrlFlag string
var bpDescFlag string
var bpCreateAccountFee string
var bpBlockSize uint32
var bpVoteCancel bool
var bpEnableCancel bool
var proposedStaminaFree uint64
var bpUpdateStaminaFree uint64
var tpsExpected uint64
var bpUpdateTpsExpected uint64
var bpUpdateCreateAccountFee string
var bpEpochDuration uint64
var bpTopN uint32
var bpPerTicketPrice string
var bpPerTicketWeight uint64

var BpCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use: "bp",
	}

	registerCmd := &cobra.Command{
		Use:     "register",
		Short:   "register as a new block-producer",
		Example: "bp register [bpname] [pubkey]",
		Args:    cobra.ExactArgs(2),
		Run:     registerBP,
	}

	registerCmd.Flags().StringVarP(&bpUrlFlag, "url", "u", "", `bp register alice --url "http://example.com"`)
	registerCmd.Flags().StringVarP(&bpDescFlag, "desc", "d", "", `bp register alice --desc "Hello World"`)
	registerCmd.Flags().StringVarP(&bpCreateAccountFee, "fee", "f", utils.MinimumCos, `bp register alice --fee 1.000000`)
	registerCmd.Flags().Uint32VarP(&bpBlockSize, "blocksize", "b", 1024*1024, `bp register alice --blocksize 1024`)
	registerCmd.Flags().Uint64VarP(&proposedStaminaFree, "stamina_free", "s", constants.DefaultStaminaFree, `bp register alice --stamina_free 1`)
	registerCmd.Flags().Uint64VarP(&tpsExpected, "tps", "t", constants.DefaultTPSExpected, `bp register alice --tps 1`)
	registerCmd.Flags().Uint64VarP(&bpEpochDuration, "epoch_duration", "", constants.InitEpochDuration, `bp register alice --epoch_duration 1000000`)
	registerCmd.Flags().Uint32VarP(&bpTopN, "top_n", "", constants.InitTopN, `bp register alice --top_n 1000`)
	registerCmd.Flags().StringVarP(&bpPerTicketPrice, "ticket_price", "", constants.PerTicketPriceStr, `bp register alice --ticket_price 5.000000`)
	registerCmd.Flags().Uint64VarP(&bpPerTicketWeight, "ticket_weight", "", constants.PerTicketWeight, `bp register alice --ticket_weight 10000000`)


	enableCmd := &cobra.Command{
		Use:     "enable",
		Short:   "enable a block-producer",
		Example: "bp enable [bpname]",
		Args:    cobra.ExactArgs(1),
		Run:     enableBP,
	}

	enableCmd.Flags().BoolVarP(&bpEnableCancel, "cancel", "c", false, `bp enable alice --cancel`)

	voteCmd := &cobra.Command{
		Use:     "vote",
		Short:   "vote to a block-producer or unvote it",
		Example: "bp vote [voter] [bpname]",
		Args:    cobra.ExactArgs(2),
		Run:     voteBp,
	}

	voteCmd.Flags().BoolVarP(&bpVoteCancel, "cancel", "c", false, `bp vote alice bob --cancel`)

	updateCmd := &cobra.Command{
		Use:     "update",
		Short:   "update block-producer state",
		Example: "bp update [bpname] --xxx xx",
		Args:    cobra.ExactArgs(1),
		Run:     updateBp,
	}

	updateCmd.Flags().Uint64VarP(&bpUpdateStaminaFree, "stamina_free", "s", bpUpdateInvalidUint64, `bp update alice --stamina_free 1`)
	updateCmd.Flags().Uint64VarP(&bpUpdateTpsExpected, "tps", "t", bpUpdateInvalidUint64, `bp update alice --tps 1`)
	updateCmd.Flags().StringVarP(&bpUpdateCreateAccountFee, "fee", "f", bpUpdateInvalidString, `bp update alice --fee 1`)
	updateCmd.Flags().Uint64VarP(&bpEpochDuration, "epoch_duration", "", bpUpdateInvalidUint64, `bp update alice --epoch_duration 1000000`)
	updateCmd.Flags().Uint32VarP(&bpTopN, "top_n", "", bpUpdateInvalidUint32, `bp update alice --top_n 1000`)
	updateCmd.Flags().StringVarP(&bpPerTicketPrice, "ticket_price", "", bpUpdateInvalidString, `bp update alice --ticket_price 5.000000`)
	updateCmd.Flags().Uint64VarP(&bpPerTicketWeight, "ticket_weight", "", bpUpdateInvalidUint64, `bp update alice --ticket_weight 10000000`)

	//updateCmd.Flags().Uint64VarP(&bpUpdateStaminaFree, "stamina_free", "s", constants.DefaultStaminaFree, `bp update alice --stamina_free 1`)
	//updateCmd.Flags().Uint64VarP(&bpUpdateTpsExpected, "tps", "t", constants.DefaultTPSExpected, `bp update alice --tps 1`)
	//updateCmd.Flags().StringVarP(&bpUpdateCreateAccountFee, "fee", "f", utils.MinimumCos, `bp update alice --fee 1`)
	//updateCmd.Flags().Uint64VarP(&bpEpochDuration, "epoch_duration", "", constants.InitEpochDuration, `bp update alice --epoch_duration 1000000`)
	//updateCmd.Flags().Uint32VarP(&bpTopN, "top_n", "", constants.InitTopN, `bp update alice --top_n 1000`)
	//updateCmd.Flags().StringVarP(&bpPerTicketPrice, "ticket_price", "", constants.PerTicketPriceStr, `bp update alice --ticket_price 5.000000`)
	//updateCmd.Flags().Uint64VarP(&bpPerTicketWeight, "ticket_weight", "", constants.PerTicketWeight, `bp update alice --ticket_weight 10000000`)

	cmd.AddCommand(registerCmd)
	cmd.AddCommand(enableCmd)
	cmd.AddCommand(voteCmd)
	cmd.AddCommand(updateCmd)
	utils.ProcessEstimate(cmd)
	return cmd
}

func registerBP(cmd *cobra.Command, args []string) {
	defer func() {
		// reset to default value
		// it's hard to assign default value from cobra.command
		// so I have to do it manually
		bpCreateAccountFee = utils.MinimumCos
		bpBlockSize = 1024 * 1024
		bpUrlFlag = ""
		bpDescFlag = ""
		proposedStaminaFree = constants.DefaultStaminaFree
		tpsExpected = constants.DefaultTPSExpected
		bpEpochDuration = constants.InitEpochDuration
		bpTopN = constants.InitTopN
		bpPerTicketPrice = constants.PerTicketPriceStr
		bpPerTicketWeight = constants.PerTicketWeight
		utils.EstimateStamina = false
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	name := args[0]
	pubKeyStr := args[1]
	bpAccount, ok := mywallet.GetUnlockedAccount(name)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", name))
		return
	}
	pubKey, err := prototype.PublicKeyFromWIF(pubKeyStr)
	if err != nil {
		fmt.Println(err)
		return
	}

	fee,err := utils.ParseCos(bpCreateAccountFee)
	if err != nil {
		fmt.Println(err)
		return
	}

	ticketPrice, err := utils.ParseCos(bpPerTicketPrice)
	if err != nil {
		fmt.Println(err)
		return
	}

	bpRegister_op := &prototype.BpRegisterOperation{
		Owner:           &prototype.AccountName{Value: name},
		Url:             bpUrlFlag,
		Desc:            bpDescFlag,
		BlockSigningKey: pubKey,
		Props: &prototype.ChainProperties{
			AccountCreationFee: prototype.NewCoin(fee),
			MaximumBlockSize:   bpBlockSize,
			StaminaFree:        proposedStaminaFree,
			TpsExpected:        tpsExpected,
			EpochDuration:      bpEpochDuration,
			TopNAcquireFreeToken: bpTopN,
			PerTicketPrice:     prototype.NewCoin(ticketPrice),
			PerTicketWeight:    bpPerTicketWeight,
		},
	}

	signTx, err := utils.GenerateSignedTxAndValidate(cmd, []interface{}{bpRegister_op}, bpAccount)
	if err != nil {
		fmt.Println(err)
		return
	}
	if utils.EstimateStamina {
		req := &grpcpb.EsimateRequest{Transaction:signTx}
		res,err := client.EstimateStamina(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(res.Invoice)
		}
	} else {
		req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
		resp, err := client.BroadcastTrx(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(fmt.Sprintf("Result: %v", resp))
		}
	}
}

func enableBP(cmd *cobra.Command, args []string) {
	defer func() {
		bpEnableCancel = false
		utils.EstimateStamina = false
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	name := args[0]
	bpAccount, ok := mywallet.GetUnlockedAccount(name)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", name))
		return
	}
	bpEnable_op := &prototype.BpEnableOperation{
		Owner:      &prototype.AccountName{Value: name},
		Cancel:     bpEnableCancel,
	}

	signTx, err := utils.GenerateSignedTxAndValidate(cmd, []interface{}{bpEnable_op}, bpAccount)
	if err != nil {
		fmt.Println(err)
		return
	}

	if utils.EstimateStamina {
		req := &grpcpb.EsimateRequest{Transaction:signTx}
		res,err := client.EstimateStamina(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(res.Invoice)
		}
	} else {
		req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
		resp, err := client.BroadcastTrx(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(fmt.Sprintf("Result: %v", resp))
		}
	}
}

func voteBp(cmd *cobra.Command, args []string) {
	defer func() {
		bpVoteCancel = false
		utils.EstimateStamina = false
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	voter := args[0]
	voterAccount, ok := mywallet.GetUnlockedAccount(voter)
	bp := args[1]
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", voter))
		return
	}
	bpVote_op := &prototype.BpVoteOperation{
		Voter:   &prototype.AccountName{Value: voter},
		BlockProducer: &prototype.AccountName{Value: bp},
		Cancel:  bpVoteCancel,
	}

	signTx, err := utils.GenerateSignedTxAndValidate(cmd, []interface{}{bpVote_op}, voterAccount)
	if err != nil {
		fmt.Println(err)
		return
	}

	if utils.EstimateStamina {
		req := &grpcpb.EsimateRequest{Transaction:signTx}
		res,err := client.EstimateStamina(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(res.Invoice)
		}
	} else {
		req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
		resp, err := client.BroadcastTrx(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(fmt.Sprintf("Result: %v", resp))
		}
	}
}

func updateBp(cmd *cobra.Command, args []string) {
	defer func() {
		//bpUpdateStaminaFree      = constants.DefaultStaminaFree
		//bpUpdateTpsExpected      = constants.DefaultTPSExpected
		//bpUpdateCreateAccountFee = utils.MinimumCos
		//bpEpochDuration = constants.InitEpochDuration
		//bpTopN = constants.InitTopN
		//bpPerTicketPrice = constants.PerTicketPriceStr
		//bpPerTicketWeight = constants.PerTicketWeight
		bpUpdateStaminaFree      = bpUpdateInvalidUint64
		bpUpdateTpsExpected      = bpUpdateInvalidUint64
		bpUpdateCreateAccountFee = bpUpdateInvalidString
		bpEpochDuration          = bpUpdateInvalidUint64
		bpTopN                   = bpUpdateInvalidUint32
		bpPerTicketPrice         = bpUpdateInvalidString
		bpPerTicketWeight        = bpUpdateInvalidUint64
		utils.EstimateStamina    = false
	}()
	c := cmd.Context["rpcclient"]
	client := c.(grpcpb.ApiServiceClient)
	w := cmd.Context["wallet"]
	mywallet := w.(wallet.Wallet)
	name := args[0]

	bpInfoOnChain, err := getBpInformation(client, name)
	if err != nil {
		fmt.Println(err)
		return
	}

	bpAccount, ok := mywallet.GetUnlockedAccount(name)
	if !ok {
		fmt.Println(fmt.Sprintf("account: %s should be loaded or created first", name))
		return
	}

	bpUpdate_op := &prototype.BpUpdateOperation{
		Owner:                 &prototype.AccountName{Value: name},
	}

	err = checkAndUpdateOpParam(bpUpdate_op, bpInfoOnChain)
	if err != nil {
		fmt.Println(err)
		return
	}

	signTx, err := utils.GenerateSignedTxAndValidate(cmd, []interface{}{bpUpdate_op}, bpAccount)
	if err != nil {
		fmt.Println(err)
		return
	}

	if utils.EstimateStamina {
		req := &grpcpb.EsimateRequest{Transaction:signTx}
		res,err := client.EstimateStamina(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(res.Invoice)
		}
	} else {
		req := &grpcpb.BroadcastTrxRequest{Transaction: signTx}
		resp, err := client.BroadcastTrx(context.Background(), req)
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println(fmt.Sprintf("Result: %v", resp))
		}
	}
}

func getBpInformation(rpcClient grpcpb.ApiServiceClient, name string) (*grpcpb.BlockProducerResponse, error) {
	req := &grpcpb.GetBlockProducerByNameRequest{BpName: &prototype.AccountName{Value: name}}
	resp, err := rpcClient.GetBlockProducerByName(context.Background(), req)
	return resp, err
}

func checkAndUpdateOpParam(op *prototype.BpUpdateOperation, infoOnChain *grpcpb.BlockProducerResponse) error {
	if bpUpdateStaminaFree == bpUpdateInvalidUint64 {
		op.ProposedStaminaFree = infoOnChain.ProposedStaminaFree
	} else {
		op.ProposedStaminaFree = bpUpdateStaminaFree
	}

	if bpUpdateTpsExpected == bpUpdateInvalidUint64 {
		op.TpsExpected = infoOnChain.TpsExpected
	} else {
		op.TpsExpected = bpUpdateTpsExpected
	}

	if bpEpochDuration == bpUpdateInvalidUint64 {
		op.EpochDuration = infoOnChain.TicketFlushInterval
	} else {
		op.EpochDuration = bpEpochDuration
	}

	if bpTopN == bpUpdateInvalidUint32 {
		op.TopNAcquireFreeToken = infoOnChain.TopNAcquireFreeToken
	} else {
		op.TopNAcquireFreeToken = bpTopN
	}

	if bpPerTicketWeight == bpUpdateInvalidUint64 {
		op.PerTicketWeight = infoOnChain.PerTicketWeight
	} else {
		op.PerTicketWeight = bpPerTicketWeight
	}

	if bpUpdateCreateAccountFee == bpUpdateInvalidString {
		op.AccountCreationFee = infoOnChain.AccountCreateFee
	} else {
		fee,err := utils.ParseCos(bpUpdateCreateAccountFee)
		if err != nil {
			fmt.Println(err)
			return err
		}
		op.AccountCreationFee = prototype.NewCoin(fee)
	}

	if bpPerTicketPrice == bpUpdateInvalidString {
		op.PerTicketPrice = infoOnChain.PerTicketPrice
	} else {
		ticketPrice, err := utils.ParseCos(bpPerTicketPrice)
		if err != nil {
			fmt.Println(err)
			return err
		}
		op.PerTicketPrice = prototype.NewCoin(ticketPrice)
	}

	return nil
}
