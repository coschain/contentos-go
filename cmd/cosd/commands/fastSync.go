package commands

import (
	"fmt"
	"github.com/coschain/cobra"
	"github.com/coschain/contentos-go/cmd/databackup/commands"
	"github.com/coschain/contentos-go/common"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

var fullNode bool
var downFileName string

var FastSyncCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:       "fast-sync",
		Short:     "fast sync mainnet data",
		Example:   "bp enable [bpname]",
		Run:       syncMainnetData,
	}
	cmd.Flags().StringVarP(&cfgName, "name", "n", "", "node name (default is cosd)")
	cmd.Flags().BoolVarP(&fullNode, "fullNode", "f", false, "start a full node or not")
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
	cmdStr := fmt.Sprintf("cd %s;rm -rf | `ls | grep -v \"config.toml\"`", dest)
	bashCmd := exec.Command("/bin/bash","-c", cmdStr)
	if err := bashCmd.Run(); err != nil {
		common.Fatalf("failed to delete old data file %v", err)
	}

	// download file from s3
	err := downloadFromS3()
	if err != nil {
		common.Fatalf("download From S3 error %v", err)
	}

	// decompress
	cmdStr = fmt.Sprintf("tar -zxvf %s -C %s", downFileName, dest)
	bashCmd = exec.Command("/bin/bash","-c", cmdStr)
	if err := bashCmd.Run(); err != nil {
		common.Fatalf("failed to decompress data file %v", err)
	}
}

func downloadFromS3() error {
	s, err := session.NewSession(&aws.Config{Region: aws.String(commands.S3_REGION)})
	if err != nil {
		return err
	}

	if fullNode {
		downFileName = commands.FULL_NODE_ARC_FILENAME
	} else {
		downFileName = commands.NON_FULL_NODE_ARC_FILENAME
	}

	file, err := os.Create(downFileName)
	if err != nil {
		return err
	}
	defer file.Close()

	downloader := s3manager.NewDownloader(s)
	_, err = downloader.Download(file,
		&s3.GetObjectInput{
			Bucket:   aws.String(commands.S3_BUCKET),
			Key:      aws.String(downFileName),
		})
	return err
}