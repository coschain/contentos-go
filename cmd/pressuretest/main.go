package main

import (
	"fmt"
	"github.com/coschain/contentos-go/cmd/pressuretest/request"
	"github.com/coschain/contentos-go/common"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func main() {

	if len(os.Args) != 7 && len(os.Args) != 8 {
		fmt.Println("params count error\n Example: pressuretest chain thread-count basename accountName publickey privateKey file-path")
		return
	}

	request.ChainId.Value = common.GetChainIdByName(os.Args[1])

	walletCnt, err := strconv.Atoi(os.Args[2])
	if err != nil {
		fmt.Println("param error: ", err)
		return
	}
	fmt.Println("robot count: ", walletCnt)

	// create 9 accounts [accountName]1 ... [accountName]9 and initminer post 10 articles
	request.InitEnv( os.Args[3], os.Args[4], os.Args[5], os.Args[6])
	fmt.Println("init base enviroment over")

	for i:=0;i<walletCnt;i++ {
		request.Wg.Add(1)
		go request.StartEachRoutine(i)
	}

	if len(os.Args) == 8 {
		request.Wg.Add(1)
		go request.StartBPRoutine()
	}

	go func() {
		SIGSTOP := syscall.Signal(0x13) //for windows compile
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
		for {
			s := <-sigc
			fmt.Printf("get a signal %s\n", s.String())
			switch s {
			case syscall.SIGQUIT, syscall.SIGTERM, SIGSTOP, syscall.SIGINT:
				request.Mu.Lock()
				request.StopSig = true
				request.Mu.Unlock()
				return
			default:
				return
			}
		}
	}()

	request.Wg.Wait()
	fmt.Println("robot exit")
}