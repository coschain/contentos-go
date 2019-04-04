package request

import (
	"fmt"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/rpc"
	"github.com/coschain/contentos-go/rpc/pb"
	"math/rand"
	"os"
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
	REPLY_CMD    = "reply"

	INIT_ACCOUNT_LENGTH = 10
	INIT_POSTID_LENGTH  = 10
	MAX_ACCOUNT_NUM     = 10000000  // 10 million
	MAX_POSTID_NUM      = 10000000  // 10 million
)

var IPList []string = []string{
	//"34.200.75.66:8888",
	//"34.195.63.116:8888",
	//"34.193.131.213:8888",
	//"34.193.58.34:8888",
	"127.0.0.1:8888",
}

var CmdTypeList []string = []string{
	CREATE_CMD,
	TRANSFER_CMD,
	POST_CMD ,
	FOLLOW_CMD,
	VOTE_CMD,
	REPLY_CMD,
}

var GlobalAccountLIst accountList
var PostIdList IdList

var Wg = &sync.WaitGroup{}

var Mu = &sync.RWMutex{}
var StopSig = false

func InitEnv( baseName string, accountName string, publicKey string, privKey string, ) {
	obj := &wallet.PrivAccount{
		Account: wallet.Account{Name: accountName, PubKey: publicKey},
		PrivKey: privKey,
	}
	GlobalAccountLIst.arr = append(GlobalAccountLIst.arr, obj)


	localWallet := wallet.NewBaseWallet("default", "" )
	conn, err := rpc.Dial( IPList[0] )
	defer conn.Close()
	if err != nil {
		common.Fatalf("Chain should have been run first")
	}
	rpcClient := grpcpb.NewApiServiceClient(conn)

	for i:=1;i<=INIT_ACCOUNT_LENGTH-1;i++ {
		createAccount(localWallet, rpcClient, GlobalAccountLIst.arr[0], fmt.Sprintf("%s%d", baseName, i))
	}
	if len(GlobalAccountLIst.arr) < INIT_ACCOUNT_LENGTH {
		fmt.Println("init account list failed, account list length: ", len(GlobalAccountLIst.arr))
		os.Exit(1)
	}

	for i:=1;i<=INIT_POSTID_LENGTH;i++ {
		postArticle(rpcClient, GlobalAccountLIst.arr[0])
	}
	if len(PostIdList.arr) < INIT_POSTID_LENGTH {
		fmt.Println("init postid list failed, postid length: ", len(PostIdList.arr))
		os.Exit(1)
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
			GlobalAccountLIst.RLock()
			if len(GlobalAccountLIst.arr) > MAX_ACCOUNT_NUM {
				GlobalAccountLIst.RUnlock()
				continue
			}
			GlobalAccountLIst.RUnlock()
			createAccount(localWallet, rpcClient, nil, "")
		case TRANSFER_CMD:
			transfer(rpcClient, nil, nil, 0)
		case POST_CMD:
			PostIdList.RLock()
			if len(PostIdList.arr) > MAX_POSTID_NUM {
				PostIdList.RUnlock()
				continue
			}
			PostIdList.RUnlock()
			postArticle(rpcClient, nil)
		case FOLLOW_CMD:
			follow(rpcClient, nil, nil)
		case VOTE_CMD:
			voteArticle(rpcClient, nil, 0)
		case REPLY_CMD:
			//replyArticle(rpcClient, nil, 0)
		}
	}
}