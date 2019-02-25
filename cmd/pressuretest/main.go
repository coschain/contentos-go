package main

import (
	"fmt"
	"os"

	"github.com/coschain/cobra"
)

var IPList []string = []string{}

var rootCmd = &cobra.Command{
	Short: "pressuretest is a pressure test client",
}

func main() {
	walletCnt := os.Args[1]
	fmt.Println("wallet count: ", walletCnt)
}