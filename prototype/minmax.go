package prototype

import (
	"github.com/coschain/contentos-go/common/constants"
	"math"
	"strings"
)

var (
	MaxAccountName = NewAccountName(strings.Repeat("z", constants.MaxAccountNameLength + 1));
	//MinAccountName = NewAccountName("")
	MinAccountName = NewAccountName( strings.Repeat("0", constants.MinAccountNameLength) )

	MaxCoin = NewCoin(math.MaxUint64);
	MinCoin = NewCoin(0);

	MaxVest = NewVest(math.MaxUint64);
	MinVest = NewVest(0);

	MaxTimePointSec = NewTimePointSec(math.MaxUint32);
	MinTimePointSec = NewTimePointSec(0);
)
