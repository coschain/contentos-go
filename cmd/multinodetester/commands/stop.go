package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"os/exec"
)

var StopCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop",
		Short: "force kill all cosd process ",
		Run:   stop,
	}
	return cmd
}

func stop(cmd *cobra.Command, args []string) {
	fmt.Println("pkill -9 cosd " )
	c   := exec.Command("pkill", "-9", "cosd")
	err := c.Run()
	if err != nil{
		fmt.Println(err)
		return
	}
}
