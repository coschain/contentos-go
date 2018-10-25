package main

import (
	"fmt"
	log "github.com/inconshreveable/log15"
	"github.com/spf13/cobra"
	"os"
	"os/signal"
	"syscall"
)

const (
	clientIdentifier = "cosd"
)

// NO OTHER CONFIG HERE EXCEPT NODE CONFIG
func cmdRunNode(cmd *cobra.Command, args []string) {
	// _ is cfg as below process has't used
	node, _ := makeConfig()
	if err := node.Start(); err != nil {
		fmt.Println("Fatal: ", err)
		os.Exit(1)
	}

	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		log.Info("Got interrupt, shutting down...")
		go node.Stop()
	}()
}

// cosd is the main entry point into the system if no special subcommand is pointed
// It creates a default node based on the command line arguments and runs it
// in blocking mode, waiting for it to be shut down.
var rootCmd = &cobra.Command{
	Use:   "cosd",
	Short: "Cosd is a fast blockchain for content",
	Run:   cmdRunNode,
}

func addCommands() {

}

func main() {
	addCommands()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

/*
type gethConfig struct {
	Node node.Config
}

func main() {
	node := makeFullNode()

	fmt.Println("1")

	startNode(node)

	fmt.Println("2")

	node.Wait()
}

func makeFullNode() *node.Node {
	stack, _ := makeConfigNode()

	return stack
}

func makeConfigNode() (*node.Node, gethConfig) {
	cfg := gethConfig{
		Node: defaultNodeConfig(),
	}

	stack, err := node.New(&cfg.Node)
	if err != nil {
		fmt.Println("Failed to create the protocol stack: %v", err)
	}

	return stack, cfg
}

func defaultNodeConfig() node.Config {
	cfg := node.DefaultConfig
	cfg.Name = clientIdentifier
	//cfg.Version = params.VersionWithCommit(gitCommit)
	return cfg
}

func startNode(stack *node.Node) {

	// Start up the node itself
	StartNode(stack)

}

func StartNode(stack *node.Node) {
	if err := stack.Start(); err != nil {
		fmt.Println("Error starting protocol stack: %v", err)
	}
	go func() {
		sigc := make(chan os.Signal, 1)
		signal.Notify(sigc, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(sigc)
		<-sigc
		fmt.Println("Got interrupt, shutting down...")
		go stack.Stop()
		for i := 10; i > 0; i-- {
			<-sigc
			if i > 1 {
				fmt.Println("Already shutting down, interrupt more to panic. ", " times: ", i-1)
			}
		}
		//debug.Exit() // ensure trace and CPU profile data is flushed.
		//debug.LoudPanic("boom")
	}()
>>>>>>> add p2p entrypoint, can listen port
}
*/
