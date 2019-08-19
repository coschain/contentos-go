package op

import (
	"github.com/coschain/contentos-go/dandelion"
	"github.com/stretchr/testify/assert"
	"testing"
)

func stakeSelf( user *dandelion.DandelionAccount, t *testing.T )  {
	a := assert.New(t)
	a.NoError( user.SendTrxAndProduceBlock( dandelion.Stake(user.Name, user.Name, 1) ) )
}