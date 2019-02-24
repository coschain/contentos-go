package utils

import (
	"fmt"
	"github.com/coschain/contentos-go/config"
	"github.com/coschain/contentos-go/node"
	"github.com/coschain/contentos-go/p2p/net/protocol"
	msgTypes "github.com/coschain/contentos-go/p2p/message/types"
	"github.com/coschain/contentos-go/p2p/net/netserver"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"testing"
)

func testHandler(data *msgTypes.MsgPayload, p2p p2p.P2P, args ...interface{}) {
	fmt.Println("Test handler")
}

// TestMsgRouter tests a basic function of a message router
func TestMsgRouter(t *testing.T) {
	conf := &config.DefaultNodeConfig
	ctx := new(node.ServiceContext)
	ctx.ResetConfig(conf)
	log := logrus.New()
	network := netserver.NewNetServer(ctx, log)
	msgRouter := NewMsgRouter(network)
	assert.NotNil(t, msgRouter)

	msgRouter.RegisterMsgHandler("test", testHandler)
	msgRouter.UnRegisterMsgHandler("test")
	msgRouter.Start()
	msgRouter.Stop()
}
