package request

import (
	"bufio"
	"fmt"
	"github.com/coschain/contentos-go/cmd/wallet-cli/wallet"
	"github.com/coschain/contentos-go/common"
	"github.com/coschain/contentos-go/common/constants"
	"github.com/coschain/contentos-go/rpc"
	"github.com/coschain/contentos-go/rpc/pb"
	"math/rand"
	"os"
	"strconv"
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

type accountInfo struct {
	name        string
	pubKeyStr   string
	priKeyStr   string
}

const (
	CREATE_CMD   = "create"
	TRANSFER_CMD = "transfer"
	POST_CMD     = "post"
	FOLLOW_CMD   = "follow"
	VOTE_CMD     = "vote"
	REPLY_CMD    = "reply"
	CONTRACT     = "contract"
	ACQUIRE_TICKET_CMD = "ticket acquire"
	VOTE_BY_TICKET_CMD = "ticket vote"

	INIT_ACCOUNT_LENGTH = 8
	INIT_POSTID_LENGTH  = 8
	MAX_ACCOUNT_NUM     = 10000000
	MAX_POSTID_NUM      = 10000000  // 10 million
)

var IPList []string = []string{
	"34.199.54.140:8888",
	"34.203.85.235:8888",
	"18.207.49.32:8888",
	"34.192.150.16:8888",
	//"127.0.0.1:8888",
}

var CmdTypeList []string = []string{
	CREATE_CMD,
	TRANSFER_CMD,
	POST_CMD ,
	FOLLOW_CMD,
	VOTE_CMD,
	REPLY_CMD,
	CONTRACT,
	ACQUIRE_TICKET_CMD,
	VOTE_BY_TICKET_CMD,
}

var GlobalAccountLIst accountList
var PostIdList IdList
var BPList []*accountInfo
var lastConductBPIndex = -1

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

	stake(rpcClient,GlobalAccountLIst.arr[0],GlobalAccountLIst.arr[0],1000000)

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
			GlobalAccountLIst.Lock()
			if len(GlobalAccountLIst.arr) > MAX_ACCOUNT_NUM {
				fmt.Println("Account list reach its lengt limit, account list length: ", len(GlobalAccountLIst.arr),
					" length limit: ", MAX_ACCOUNT_NUM,
					" timestamp: ", time.Now())
				GlobalAccountLIst.Unlock()
				continue
			}
			GlobalAccountLIst.Unlock()
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
			replyArticle(rpcClient, nil, 0)
		case CONTRACT:
			callContract(rpcClient, nil)
		case ACQUIRE_TICKET_CMD:
			acquireTicket(rpcClient, nil)
		case VOTE_BY_TICKET_CMD:
			voteByTicket(rpcClient, nil, 0)
		}
	}
}

func StartBPRoutine(){
	defer Wg.Done()

	filePath := os.Args[7]
	bpListFile, err := os.Open(filePath)
	if err != nil {
		fmt.Println("can't open BP list file: ", err)
		return
	}
	defer bpListFile.Close()

	scanner := bufio.NewScanner(bpListFile)
	scanner.Split(bufio.ScanLines)

	// read bp list file into BPList
	readBPListFile(scanner)
	fmt.Println("BP length: ", len(BPList))

	conn, err := rpc.Dial( IPList[0] )
	defer conn.Close()
	if err != nil {
		common.Fatalf("Chain should have been run first")
	}
	rpcClient := grpcpb.NewApiServiceClient(conn)

	err = InitLastConductBPIndex(rpcClient)
	if err != nil {
		fmt.Println("Get bp list on chain error: ", err)
		return
	}

	fmt.Println("lastConductBPIndex: ", lastConductBPIndex)

	for ;; {
		Mu.RLock()
		if StopSig == true {
			Mu.RUnlock()
			break
		}
		Mu.RUnlock()

		if lastConductBPIndex == -1 {
			// random disable a bp
			RandomDisableBP(rpcClient)
		} else {
			// first enable last disable bp
			// if no error
			// random disable a new bp
			err := EnableBP(rpcClient, lastConductBPIndex)
			if err != nil {
				fmt.Println("Enable BP error: ", err)
				continue
			}
			RandomDisableBP(rpcClient)
		}

		time.Sleep(time.Duration(len(BPList) * constants.BlockProdRepetition) * time.Second)
	}
}

func readBPListFile(scanner *bufio.Scanner) {
	line := 0
	for scanner.Scan() {
		line++
		mod := line % 3
		lineTextStr := scanner.Text()

		if mod == 1 {
			newBP := new(accountInfo)
			newBP.name = lineTextStr
			BPList = append(BPList, newBP)
		} else if mod == 2 {
			tail := len(BPList) - 1
			BPList[tail].pubKeyStr = lineTextStr
		} else {
			tail := len(BPList) - 1
			BPList[tail].priKeyStr = lineTextStr
		}
	}
}

func InitLastConductBPIndex(rpcClient grpcpb.ApiServiceClient) error {
	bpListOnChain, err := getBPListOnChain(rpcClient)
	if err != nil {
		return err
	}
	bpNumSum := 0
	for i:=1;i<=len(BPList);i++ {
		bpNumSum += i
	}
	if len(bpListOnChain.BlockProducerList) != len(BPList) {
		sum := 0
		for i:=0;i<len(bpListOnChain.BlockProducerList);i++ {
			if bpListOnChain.BlockProducerList[i].GetBpVest().Active {
				bpNumStr := bpListOnChain.BlockProducerList[i].Owner.Value[9:]
				bpNum, err := strconv.Atoi(bpNumStr)
				if err != nil {
					return err
				}
				sum += bpNum
			}
		}
		lastConductBPIndex = bpNumSum - sum - 1
	}
	return nil
}