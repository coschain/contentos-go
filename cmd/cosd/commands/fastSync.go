package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/databackup/commands"
	"github.com/coschain/contentos-go/common"
)

var downFileName string

var FastSyncCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "fast-sync",
		Short:     "fast sync mainnet data",
		Example:   "bp enable [bpname]",
		Run:       syncMainnetData,
	}
	cmd.Flags().StringVarP(&cfgName, "name", "n", "", "node name (default is cosd)")
	return cmd
}

func syncMainnetData(cmd *cobra.Command, args []string) {
	// read config to get data directory
	cfg := readConfig()

	// generate data directory absolute path
	if cfg.DataDir != "" {
		dir, err := filepath.Abs(cfg.DataDir)
		if err != nil {
			common.Fatalf("DataDir in cfg cannot be converted to absolute path")
		}
		cfg.DataDir = dir
	}
	dest := filepath.Join(cfg.DataDir, cfg.Name)

	// delete old data file
	cmdStr := fmt.Sprintf("cd %s;rm -rf `ls | grep -v \"config.toml\"`", dest)
	bashCmd := exec.Command("/bin/bash","-c", cmdStr)
	if err := bashCmd.Run(); err != nil {
		common.Fatalf("failed to delete old data file %v", err)
	}

	downFileName = getTargetFileName()
	if downFileName == "" {
		common.Fatalf("failed to get target file name")
	}

	// download file from s3
	cmdStr = fmt.Sprintf("wget https://%s.s3.amazonaws.com/%s", commands.S3_BUCKET, downFileName)
	bashCmd = exec.Command("/bin/bash","-c", cmdStr)
	bashCmd.Stdout = os.Stdout
	bashCmd.Stderr = os.Stderr
	if err := bashCmd.Run(); err != nil {
		common.Fatalf("failed to download data file %v", err)
	}

	// decompress
	cmdStr = fmt.Sprintf("tar -zxvf %s -C %s", downFileName, dest)
	bashCmd = exec.Command("/bin/bash","-c", cmdStr)
	bashCmd.Stdout = os.Stdout
	bashCmd.Stderr = os.Stderr
	if err := bashCmd.Run(); err != nil {
		common.Fatalf("failed to decompress data file %v", err)
	}

	// delete compressed data file and route file
	os.Remove(downFileName)
	os.Remove(commands.ROUTER)
}

func getTargetFileName() (name string) {
	// download route file
	cmdStr := fmt.Sprintf("wget https://%s.s3.amazonaws.com/%s", commands.S3_BUCKET, commands.ROUTER)
	bashCmd := exec.Command("/bin/bash","-c", cmdStr)
	bashCmd.Stdout = os.Stdout
	bashCmd.Stderr = os.Stderr
	if err := bashCmd.Run(); err != nil {
		common.Fatalf("failed to download route file %v", err)
	}

	// get target file name, aka last line content
	file, err := os.Open(commands.ROUTER)
	if err != nil {
		common.Fatalf("failed to read route file %v", err)
		return
	}

	fileScanner := bufio.NewScanner(file)
	for fileScanner.Scan(){
		name = fileScanner.Text()
	}

	return name
}