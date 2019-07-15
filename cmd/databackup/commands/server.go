package commands

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/coschain/cobra"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

var port int16
var ip string
var backupDir string

var ServerCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "start backup server",
		Run:   startBackUpServer,
	}
	cmd.Flags().Int16VarP(&port, "port", "p", 9722, "")
	cmd.Flags().StringVarP(&ip, "ip", "i", "", "")
	cmd.Flags().StringVarP(&backupDir, "dir", "d", "~/contentos_data_backup", "")
	return cmd
}

func startBackUpServer(cmd *cobra.Command, args []string) {
	router := gin.Default()
	// Set a lower memory limit for multipart forms (default is 32 MiB)
	// router.MaxMultipartMemory = 8 << 20  // 8 MiB
	router.Static("/download", backupDir)
	router.POST("/uploads", func(c *gin.Context) {
		// Multipart form
		form, _ := c.MultipartForm()
		files := form.File["file"]

		for _, file := range files {
			logrus.Debugf("recv file %s", file.Filename)
			err := c.SaveUploadedFile(file, backupDir+"/tmp/"+file.Filename)
			if err != nil {
				logrus.Error(err)
				c.String(http.StatusInternalServerError, fmt.Sprintf("failed to save files: %s", err.Error()))
				return
			}
		}

		c.String(http.StatusOK, fmt.Sprintf("%d files uploaded!", len(files)))
	})
	router.POST("/upload", func(c *gin.Context) {
		// single file
		file, err := c.FormFile("file")
		if err != nil {
			logrus.Error(err)
			c.String(http.StatusBadRequest, fmt.Sprintf("failed to retrieve files: %s", err.Error()))
			return
		}

		err = c.SaveUploadedFile(file, backupDir+"/tmp/"+file.Filename)
		if err != nil {
			logrus.Error(err)
			c.String(http.StatusInternalServerError, fmt.Sprintf("failed to save files: %s", err.Error()))
		}
		c.String(http.StatusOK, fmt.Sprintf("'%s' uploaded!", file.Filename))
	})
	router.Run(ip+":"+strconv.Itoa(int(port)))
}