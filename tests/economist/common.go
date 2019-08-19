package economist

import (
	"github.com/coschain/contentos-go/common/constants"
	. "github.com/coschain/contentos-go/dandelion"
	"github.com/coschain/contentos-go/prototype"
	"github.com/stretchr/testify/assert"
	"testing"
)

var mintProps = &prototype.ChainProperties{
	AccountCreationFee: prototype.NewCoin(1),
	StaminaFree:        constants.DefaultStaminaFree,
	TpsExpected:        constants.DefaultTPSExpected,
	EpochDuration:      constants.InitEpochDuration,
	TopNAcquireFreeToken: constants.InitTopN,
	PerTicketPrice:     prototype.NewCoin(1000000),
	PerTicketWeight:    constants.PerTicketWeight,
}

func registerBlockProducer(account *DandelionAccount, t *testing.T)  {
	a := assert.New(t)
	a.NoError(account.SendTrxAndProduceBlock(TransferToVest(account.Name, account.Name, constants.MinBpRegisterVest, "")))
	a.NoError(account.SendTrxAndProduceBlock(BpRegister(account.Name, "", "", account.GetPubKey(), mintProps)))
}

func RegisterBlockProducer(account *DandelionAccount, t *testing.T)  {
	registerBlockProducer(account, t)
}