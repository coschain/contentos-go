package main

import (
	"bufio"
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/pressuretest/request"
	"github.com/coschain/contentos-go/common"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

var nodeFilePath, bpFilePath string

var rootCmd = &cobra.Command{
	Use:   "pressuretest",
	Run:   startTest,
}

func main() {
	rootCmd.Flags().StringVarP(&nodeFilePath, "node", "n", "", "./pressuretest -n=/filepath/to/pressuretest/nodelist")
	rootCmd.Flags().StringVarP(&bpFilePath, "bp", "p", "", "./pressuretest -p=/filepath/to/bplist")
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func startTest(cmd *cobra.Command, args []string) {

	if len(args) < 6 {
		fmt.Println("params count error\n Example: pressuretest chain thread-count basename accountName publickey privateKey")
		return
	}

	request.ChainId.Value = common.GetChainIdByName(args[0])

	walletCnt, err := strconv.Atoi(args[1])
	if err != nil {
		fmt.Println("param error: ", err)
		return
	}
	fmt.Println("robot count: ", walletCnt)

	if nodeFilePath == "" {
		request.IPList = append(request.IPList, "127.0.0.1:8888")
	} else {
		err := readNodeListFile(nodeFilePath)
		if err != nil {
			fmt.Println("can't read nodes list file: ")
			return
		}
	}

	// create 9 accounts [accountName]1 ... [accountName]9 and initminer post 10 articles
	request.InitEnv( args[2], args[3], args[4], args[5])
	fmt.Println("init base enviroment over")

	for i:=0;i<walletCnt;i++ {
		request.Wg.Add(1)
		go request.StartEachRoutine(i)
	}

	if bpFilePath != "" {
		request.Wg.Add(1)
		go request.StartBPRoutine(bpFilePath)
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

func readNodeListFile(path string) error {
	nodeListFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer nodeListFile.Close()

	scanner := bufio.NewScanner(nodeListFile)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		lineTextStr := scanner.Text()
		request.IPList = append(request.IPList, lineTextStr)
	}
	return nil
}