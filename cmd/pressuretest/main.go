package main

import (
	"fmt"
	"github.com/coschain/cobra"
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
	fmt.Println("wallet count: ", walletCnt)
	os.Args = os.Args[1:]

	request.InitEnv()

	for i:=0;i<walletCnt;i++ {
		go func(){
			rootCmd := request.MakeRootCmd()

			rootCmd.Run = func(cmd *cobra.Command, args []string) {
				request.RunShell(rootCmd)
			}

			request.MakeWallet(i, rootCmd)
		}()
	}

	SIGSTOP := syscall.Signal(0x13) //for windows compile
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)
	for {
		s := <-sigc
		fmt.Printf("get a signal %s\n", s.String())
		switch s {
		case syscall.SIGQUIT, syscall.SIGTERM, SIGSTOP, syscall.SIGINT:
			fmt.Println("Got interrupt, shutting down...")
			os.Exit(0)
			return
		default:
			return
		}
	}
}