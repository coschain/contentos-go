package main

import (
	"fmt"
	"github.com/coschain/contentos-go/cmd/pressuretest/request"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func main() {
	walletCnt, err := strconv.Atoi(os.Args[1])
	if err != nil {
		fmt.Println("param error: ", err)
		return
	}
	fmt.Println("robot count: ", walletCnt)

	// create 10 accounts initminer1 ... initminer10 and initminer post 10 articles
	request.InitEnv()
	fmt.Println("init base enviroment over")

	for i:=0;i<walletCnt;i++ {
		request.Wg.Add(1)
		go request.StartEachRoutine(i)
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