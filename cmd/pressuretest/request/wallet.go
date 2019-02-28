package request

import (
	"fmt"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/rpc"
	"github.com/coschain/contentos-go/rpc/pb"
	"math/rand"
	"sync"
	"time"
)

type accountList struct {
	sync.RWMutex
	arr  []*wallet.PrivAccount
}

type IdList struct {
	sync.RWMutex
	arr []uint64
}

const (
	CREATE_CMD   = "create"
	TRANSFER_CMD = "transfer"
	POST_CMD     = "post"
	FOLLOW_CMD   = "follow"
	VOTE_CMD     = "vote"
)

var IPList []string = []string{
	"localhost:8888",
}

var CmdTypeList []string = []string{
	CREATE_CMD,
	TRANSFER_CMD,
	POST_CMD ,
	FOLLOW_CMD,
	VOTE_CMD,
}

var GlobalAccountLIst accountList
var PostIdList IdList

var Wg = &sync.WaitGroup{}

var Mu = &sync.RWMutex{}
var StopSig = false

func InitEnv() {
	obj := &wallet.PrivAccount{
		Account: wallet.Account{Name: "initminer", PubKey: constants.InitminerPubKey},
		PrivKey: constants.InitminerPrivKey,
	}
	GlobalAccountLIst.arr = append(GlobalAccountLIst.arr, obj)


	localWallet := wallet.NewBaseWallet("default", "" )
	conn, err := rpc.Dial( IPList[0] )
	defer conn.Close()
	if err != nil {
		common.Fatalf("Chain should have been run first")
	}
	rpcClient := grpcpb.NewApiServiceClient(conn)

	for i:=1;i<=10;i++ {
		createAccount(localWallet, rpcClient, GlobalAccountLIst.arr[0], fmt.Sprintf("initminer%d", i))
	}

	for i:=1;i<=10;i++ {
		postArticle(rpcClient, GlobalAccountLIst.arr[0])
	}
}

func StartEachRoutine(index int) {
	defer Wg.Done()

	localWallet := wallet.NewBaseWallet("default", "" )
	conn, err := rpc.Dial( IPList[ index%len(IPList) ] )
	defer conn.Close()
	if err != nil {
		common.Fatalf("Chain should have been run first")
	}
	rpcClient := grpcpb.NewApiServiceClient(conn)

	for {
		Mu.RLock()
		if StopSig == true {
			Mu.RUnlock()
			break
		}
		Mu.RUnlock()

		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		idx := r.Intn(len(CmdTypeList))
		cmdType := CmdTypeList[idx]

		switch cmdType {
		case CREATE_CMD:
			createAccount(localWallet, rpcClient, nil, "")
		case TRANSFER_CMD:
			transfer(rpcClient, nil, nil, 0)
		case POST_CMD:
			postArticle(rpcClient, nil)
		case FOLLOW_CMD:
			follow(rpcClient, nil, nil)
		case VOTE_CMD:
			voteArticle(rpcClient, nil, 0)
		}
	}
}