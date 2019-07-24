package main

import (
	"flag"
	"fmt"
	"github.com/coschain/contentos-go/cmd/nodedetector/detector"
	"strings"
)

const (
	RpcPort = "8888"
)

// seed nodes of the chain, if you have multi seed nodes, please seperate ip with ","
var seednodes = flag.String("seed", "", "seed nodes list")

func main() {
	flag.Parse()

	if *seednodes == "" {
		fmt.Println("Oops your seed nodes is empty")
		return
	}

	seedNodesList := strings.Split(*seednodes, ",")

	for i:=0;i<len(seedNodesList);i++ {
		endPoint := fmt.Sprintf("%s:%s", seedNodesList[i], RpcPort)
		detector.RequireNodeInfo(endPoint)
	}
}