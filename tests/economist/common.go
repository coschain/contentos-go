package economist

import (
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/prototype"
)

var mintProps = &prototype.ChainProperties{
	AccountCreationFee: prototype.NewCoin(1),
	MaximumBlockSize:   1024 * 1024,
	StaminaFree:        constants.DefaultStaminaFree,
	TpsExpected:        constants.DefaultTPSExpected,
	EpochDuration:      constants.InitEpochDuration,
	TopNAcquireFreeToken: constants.InitTopN,
	PerTicketPrice:     prototype.NewCoin(1000000),
	PerTicketWeight:    constants.PerTicketWeight,
}
