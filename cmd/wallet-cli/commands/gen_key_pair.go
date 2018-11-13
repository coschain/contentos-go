package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/prototype"
)

var GenKeyPairCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "genKeyPair",
		Short: "generate new key pair",
		Run:   genKeyPair,
	}
	return cmd
}

func genKeyPair(cmd *cobra.Command, args []string) {

	priKey, err := prototype.GenerateNewKey()

	if err != nil {
		fmt.Println("GenerateNewKey Error: ", err)
	}

	pubKey, err := priKey.PubKey()
	if err != nil {
		fmt.Println("GeneratePubKey Error: ", err)
	}

	fmt.Println("Public  Key: ", pubKey.ToWIF() )
	fmt.Println("Private Key: ", priKey.ToWIF() )

}
