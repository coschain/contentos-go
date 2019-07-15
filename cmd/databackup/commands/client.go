package commands

import (
	"github.com/coschain/cobra"
)

var serverPort int16
var serverIP string

var ClientCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "client",
		Short: "start backup client",
		Run:   startBackUpClient,
	}
	cmd.Flags().Int16VarP(&serverPort, "server_port", "p", 9722, "")
	cmd.Flags().StringVarP(&serverIP, "server_ip", "i", "", "")
	return cmd
}

func startBackUpClient(cmd *cobra.Command, args []string) {}

// downloadFromUrl("http://localhost:8062/file/file1.bin")
/*
func downloadFromUrl(url string) {
	tokens := strings.Split(url, "/")
	fileName := tokens[len(tokens)-1]
	fmt.Println("Downloading", url, "to", fileName)

	// TODO: check file existence first with io.IsExist
	output, err := os.Create(fileName)
	if err != nil {
		fmt.Println("Error while creating", fileName, "-", err)
		return
	}
	defer output.Close()

	response, err := http.Get(url)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}
	defer response.Body.Close()

	n, err := io.Copy(output, response.Body)
	if err != nil {
		fmt.Println("Error while downloading", url, "-", err)
		return
	}

	fmt.Println(n, "bytes downloaded.")
}
*/