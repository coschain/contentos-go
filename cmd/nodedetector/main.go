package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/coschain/contentos-go/cmd/nodedetector/detector"
)

const (
	RpcPort          = "8888"
	AutoScanInterval = 5 * 60
)

// seed nodes of the chain, if you have multi seed nodes, please seperate ip with ","
var seednodes = flag.String("seed", "", "seed nodes list")

var stopSig = make(chan os.Signal, 1)

var processed map[string]struct{}

func main() {
	flag.Parse()

	if *seednodes == "" {
		fmt.Println("Oops your seed nodes is empty")
		return
	}

	nodeManager := detector.Init()
	initSignal()
	processed = make(map[string]struct{})
	seedNodesList := strings.Split(*seednodes, ",")

	nodeManager.AddToQuerylist(seedNodesList, false)

	ticker := time.NewTicker(time.Second * AutoScanInterval)
	for {
		select {
		case <-stopSig:
			ticker.Stop()
			detector.Wg.Wait()
			fmt.Println("exit safe")
			os.Exit(0)
		case ip := <-detector.QueryList:
			if !checkExist(ip) {
				addProcessed(ip)
				endPoint := fmt.Sprintf("%s:%s", ip, RpcPort)
				detector.Wg.Add(1)
				go nodeManager.Query(endPoint)
			}
		case <- ticker.C:
			detector.Wg.Wait()
			nodeManager.Reset()
			clearProcessed()
			fmt.Println("\n\n=========================================")
			fmt.Println("Start a new round to scan the whole net")
			fmt.Println("=========================================\n\n")
			nodeManager.AddToQuerylist(seedNodesList, false)
		}
	}
}

func checkExist(ip string) bool {
	_, ok := processed[ip]
	return ok
}

func addProcessed(ip string) {
	processed[ip] = struct{}{}
}

func clearProcessed() {
	processed = make(map[string]struct{})
}

func initSignal() {
	go func () {
		SIGSTOP := syscall.Signal(0x13) //for windows compile
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT, SIGSTOP, syscall.SIGUSR1, syscall.SIGUSR2)
		for {
			s := <-sigc
			fmt.Printf("get a signal %s\n", s.String())
			switch s {
			case syscall.SIGQUIT, syscall.SIGTERM, SIGSTOP, syscall.SIGINT:
				stopSig <- s
				return
			default:
				stopSig <- s
				return
			}
		}
	}()
}