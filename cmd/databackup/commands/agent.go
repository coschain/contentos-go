package commands

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/coschain/cobra"
	"github.com/sirupsen/logrus"
)

var dataDir string
var interval int32
var archFileName string
var archFileNameSuffix string
var fullNodeBackup bool
var server *http.Server
//var destAddr string

const (
	S3_REGION    = "us-east-1" // lightsail.RegionNameEuCentral1
	S3_BUCKET    = "crystalline-cosd-databackup"
	FAKE_PORT    = "9090"

	FULL_NODE_SUFFIX       = "-fulldata.tar.gz"
	NON_FULL_NODE_SUFFIX   = "-data.tar.gz"

	TMP_DIR_NAME    = "/tmp"
	BLOG_NAME       = "/blog"
	CHECHPOINT_NAME = "/checkpoint"
	DB_NAME         = "/db"

	ROUTER = "route.txt"
)

var AgentCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "start backup agent",
		Long:  "start back agent and regularly sends cosd data files to backup server",
		Run:   startBackUpAgent,
	}
	cmd.Flags().StringVarP(&dataDir, "data_dir", "d", "", "directory of cosd data")
	cmd.Flags().Int32VarP(&interval, "interval", "i", 3 * 86400, "backup data every interval seconds")
	cmd.Flags().BoolVarP(&fullNodeBackup, "fullNodeBackup", "f", false, "backup a full node or not")
	//cmd.Flags().StringVarP(&destAddr, "addr", "a", "", "the address of the backup server")
	return cmd
}

func startBackUpAgent(cmd *cobra.Command, args []string) {
	logrus.SetReportCaller(true)
	if dataDir == "" {
		logrus.Error("data_dir cannot be empty")
		return
	}

	agent := &Agent{
		stopCh: make(chan struct{}),
	}
	agent.Run()
}

type Agent struct {
	stopCh chan struct{}
}

func (a *Agent) Run() {
	if fullNodeBackup {
		archFileNameSuffix = FULL_NODE_SUFFIX
	} else {
		archFileNameSuffix = NON_FULL_NODE_SUFFIX
	}

	// init fake server handler
	http.HandleFunc("/", fakeHandler)

	a.run()
	t := time.NewTicker(time.Second * time.Duration(interval))
	for {
		select {
		case <-t.C:
			a.run()
		case <-a.stopCh:
			return
		}
	}
}

func (a *Agent) run() error {
	logrus.Info("start a new backup round")

	defer func() {
		// delete old file
		os.RemoveAll(dataDir + TMP_DIR_NAME)
		os.Remove(archFileName)
		os.Remove(ROUTER)
	}()

	timeNow := time.Now()
	timeString := timeNow.Format("20060102-150405")
	archFileName = timeString + archFileNameSuffix

	// kill the running cosd process
	cmd := exec.Command("/bin/bash","-c", "pkill cosd")
	if err := cmd.Run(); err != nil {
		logrus.Error(err)
		return err
	}

	// start a fake port to cheat the monitor
	StartFakePort()

	// copy the data file to a tmp directory
	err := CopyDataFile(dataDir)
	if err != nil {
		logrus.Error(err)
		return err
	}

	// stop fake port
	StopFakePort()

	// start real cosd process
	//exec.Command("/bin/bash","-c","/data/coschain/contentos-go/cmd/cosd/cosd start -n /data/logs/coschain/cosd/")
	cmd = exec.Command("/bin/bash","-c","/data/coschain/src/deploy/boot.sh")
	if err := cmd.Start(); err != nil {
		logrus.Error(err)
		return err
	}

	// compress data file in tmp directory
	if err := zip(); err == nil {
		logrus.Info("cosd data archived at ", time.Now())
	} else {
		logrus.Error(err)
		return err
	}

	if err := SendToS3(); err == nil {
		logrus.Info("cosd data uploaded at ", time.Now())
	} else {
		logrus.Error(err)
		return err
	}

	return nil
}

func zip() error {
	input := make([]*os.File, 3)
	inputName := []string{
		dataDir + TMP_DIR_NAME + BLOG_NAME,
		dataDir + TMP_DIR_NAME + CHECHPOINT_NAME,
		dataDir + TMP_DIR_NAME + DB_NAME,
	}
	for i := range inputName {
		dataFile, err := os.Open(inputName[i])
		if err != nil {
			logrus.Error(err)
			return err
		}
		input[i] = dataFile
	}

	err := Compress(input, archFileName)
	if err != nil {
		logrus.Error(err)
		return err
	}
	return nil
}


//func Upload(url, file string) (err error) {
//	// Prepare a form that you will submit to that URL.
//	var b bytes.Buffer
//	w := multipart.NewWriter(&b)
//	// Add your image file
//	f, err := os.Open(file)
//	if err != nil {
//		return
//	}
//	defer f.Close()
//	fw, err := w.CreateFormFile("file", file)
//	if err != nil {
//		return
//	}
//	if _, err = io.Copy(fw, f); err != nil {
//		return
//	}
//
//	// Add the other fields
//	if fw, err = w.CreateFormField("key"); err != nil {
//		return
//	}
//	if _, err = fw.Write([]byte("KEY")); err != nil {
//		return
//	}
//	// Don't forget to close the multipart writer.
//	// If you don't close it, your request will be missing the terminating boundary.
//	w.Close()
//
//	// Now that you have a form, you can submit it to your handler.
//	req, err := http.NewRequest("POST", url, &b)
//	if err != nil {
//		return
//	}
//	// Don't forget to set the content type, this will contain the boundary.
//	req.Header.Set("Content-Type", w.FormDataContentType())
//
//	// Submit the request
//	client := &http.Client{}
//	res, err := client.Do(req)
//	if err != nil {
//		return
//	}
//
//	// Check the response
//	if res.StatusCode != http.StatusOK {
//		err = fmt.Errorf("bad status: %s", res.Status)
//	}
//	return
//}
//
//func UploadTwo() (err error) {
//	url := "http://localhost:8062/many"
//	// Prepare a form that you will submit to that URL.
//	var b bytes.Buffer
//	w := multipart.NewWriter(&b)
//
//	for i := 1; i <= 100; i++ {
//		fname := fmt.Sprintf("file%d.bin", i)
//		fw, err2 := w.CreateFormFile("file", fname)
//		if err2 != nil {
//			err = err2
//			return
//		}
//
//		f, err2 := os.Open(fname)
//		if err2 != nil {
//			err = err2
//			return
//		}
//		if _, err2 = io.Copy(fw, f); err2 != nil {
//			err = err2
//			return
//		}
//		f.Close()
//	}
//
//	// Add the other fields
//	fw, err2 := w.CreateFormField("key")
//	if err2 != nil {
//		err = err2
//		return
//	}
//	if _, err2 = fw.Write([]byte("KEY")); err2 != nil {
//		err = err2
//		return
//	}
//
//	// Don't forget to close the multipart writer.
//	// If you don't close it, your request will be missing the terminating boundary.
//	w.Close()
//
//	// Now that you have a form, you can submit it to your handler.
//	req, err := http.NewRequest("POST", url, &b)
//	if err != nil {
//		return
//	}
//	// Don't forget to set the content type, this will contain the boundary.
//	req.Header.Set("Content-Type", w.FormDataContentType())
//
//	// Submit the request
//	client := &http.Client{}
//	res, err := client.Do(req)
//	if err != nil {
//		return
//	}
//
//	// Check the response
//	if res.StatusCode != http.StatusOK {
//		err = fmt.Errorf("bad status: %s", res.Status)
//	}
//	return
//}


func Compress(files []*os.File, dest string) error {
	d, _ := os.Create(dest)
	defer d.Close()
	gw := gzip.NewWriter(d)
	defer gw.Close()
	tw := tar.NewWriter(gw)
	defer tw.Close()
	for _, file := range files {
		//logrus.Infof("name = %s", file.Name())
		err := compress(file, "", tw)
		if err != nil {
			return err
		}
	}
	return nil
}

func compress(file *os.File, prefix string, tw *tar.Writer) error {
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if info.IsDir() {
		prefix = prefix + "/" + info.Name()
		fileInfos, err := file.Readdir(-1)
		if err != nil {
			return err
		}
		for _, fi := range fileInfos {
			f, err := os.Open(file.Name() + "/" + fi.Name())
			if err != nil {
				return err
			}
			err = compress(f, prefix, tw)
			if err != nil {
				return err
			}
		}
	} else {
		if !info.Mode().IsRegular() {
			logrus.Warnf("skip file %s coz it's irregular", info.Name())
			return nil
		}
		header, err := tar.FileInfoHeader(info, "")
		header.Name = prefix + "/" + header.Name
		if err != nil {
			return err
		}
		err = tw.WriteHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(tw, file)
		file.Close()
		if err != nil {
			return err
		}

		//names := strings.Split(prefix, "/")
		//logrus.Infof("compress %s %s done", names[len(names)-1], info.Name())
	}
	return nil
}

func DeCompress(tarFile, dest string) error {
	srcFile, err := os.Open(tarFile)
	if err != nil {
		return err
	}
	defer srcFile.Close()
	gr, err := gzip.NewReader(srcFile)
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			} else {
				return err
			}
		}
		filename := dest + hdr.Name
		file, err := createFile(filename)
		if err != nil {
			return err
		}
		io.Copy(file, tr)
	}
	return nil
}

func createFile(name string) (*os.File, error) {
	err := os.MkdirAll(string([]rune(name)[0:strings.LastIndex(name, "/")]), 0755)
	if err != nil {
		return nil, err
	}
	return os.Create(name)
}

func SendToS3() error {
	// Create a single AWS session (we can re use this if we're uploading many files)
	s, err := session.NewSession(&aws.Config{Region: aws.String(S3_REGION)})
	if err != nil {
		return err
	}

	// download router and update it
	err = UpdateRouter(s, archFileName)
	if err != nil {
		return err
	}


	// Upload
	err = AddFileToS3(s, ROUTER)
	if err != nil {
		return err
	}

	err = AddFileToS3(s, archFileName)
	if err != nil {
		return err
	}
	return nil
}

// AddFileToS3 will upload a single file to S3, it will require a pre-built aws session
// and will set file info like content type and encryption on the uploaded file.
func AddFileToS3(s *session.Session, fileDir string) error {
	// Open the file for use
	file, err := os.Open(fileDir)
	if err != nil {
		return err
	}
	defer file.Close()

	// Get file size and read the file content into a buffer
	//fileInfo, _ := file.Stat()
	//var size int64 = fileInfo.Size()
	//buffer := make([]byte, size)
	//file.Read(buffer)

	// Config settings: this is where you choose the bucket, filename, content-type etc.
	// of the file you're uploading.

	uploader := s3manager.NewUploader(s)
	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket:               aws.String(S3_BUCKET),
		Key:                  aws.String(fileDir),
		Body:                 file,
		ACL:                  aws.String("public-read"),
	},func(u *s3manager.Uploader) {
		u.PartSize = 1024 * 1024 * 1024 // size of chunk  1GB
		u.LeavePartsOnError = true
		u.Concurrency = 5})
	if err != nil {
		logrus.Error("upload to s3 error ", err)
		os.Exit(-1)
	}

	//svc := s3.New(s)
	//_, err = svc.PutObject(&s3.PutObjectInput{
	//	Bucket:               aws.String(S3_BUCKET),
	//	Key:                  aws.String(fileDir),
	//	//ACL:                  aws.String("private"),
	//	Body:                 file,//bytes.NewReader(buffer),
	//	//ContentLength:        aws.Int64(size),
	//	//ContentType:          aws.String(http.DetectContentType(buffer)),
	//	//ContentDisposition:   aws.String("attachment"),
	//	//ServerSideEncryption: aws.String("AES256"),
	//})
	//
	//req, _ := svc.GetObjectRequest(&s3.GetObjectInput{
	//	Bucket: aws.String(S3_BUCKET),
	//	Key:    aws.String(fileDir),
	//})
	//urlStr, err := req.Presign(24 * time.Hour)
	//if err != nil {
	//	logrus.Println("Failed to sign request", err)
	//}
	//logrus.Info("presigned URL: ", urlStr)
	return nil
}

func UpdateRouter(s *session.Session, content string) error {
	file, err := os.Create(ROUTER)
	if err != nil {
		return err
	}
	defer file.Close()

	downloader := s3manager.NewDownloader(s)
	_, err = downloader.Download(file,
		&s3.GetObjectInput{
			Bucket: aws.String(S3_BUCKET),
			Key:    aws.String(ROUTER),
		})
	if err != nil {
		return err
	}

	writeFd, err := os.OpenFile(ROUTER, os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		panic(err)
	}
	defer writeFd.Close()

	writer := bufio.NewWriter(writeFd)
	fmt.Fprintln(writer, content)
	writer.Flush()

	return nil
}

func CopyDataFile(prefix string) error {
	// create tmp directory
	err := os.Mkdir(prefix+TMP_DIR_NAME, os.ModePerm)
	if err != nil{
		return errors.New(fmt.Sprintf("Failed to create tmp directory %s", err))
	}

	// copy blog
	cmdStr := fmt.Sprintf("cp -r %s %s", prefix+BLOG_NAME, prefix+TMP_DIR_NAME+BLOG_NAME)
	cmd := exec.Command("/bin/bash","-c", cmdStr)
	if err := cmd.Run(); err != nil {
		return errors.New(fmt.Sprintf("failed to copy blog %s", err))
	}

	// copy checkpoint
	cmdStr = fmt.Sprintf("cp -r %s %s", prefix+CHECHPOINT_NAME, prefix+TMP_DIR_NAME+CHECHPOINT_NAME)
	cmd = exec.Command("/bin/bash","-c", cmdStr)
	if err := cmd.Run(); err != nil {
		return errors.New(fmt.Sprintf("failed to copy checkpoint %s", err))
	}

	// copy db
	cmdStr = fmt.Sprintf("cp -r %s %s", prefix+DB_NAME, prefix+TMP_DIR_NAME+DB_NAME)
	cmd = exec.Command("/bin/bash","-c", cmdStr)
	if err := cmd.Run(); err != nil {
		return errors.New(fmt.Sprintf("failed to copy db %s", err))
	}

	return nil
}

func StartFakePort() {
	server = &http.Server{Addr: fmt.Sprintf(":%s", FAKE_PORT)}
	go server.ListenAndServe()
}

func StopFakePort() {
	ctx, _ := context.WithTimeout(context.Background(), 5 * time.Second)
	if err := server.Shutdown(ctx); err != nil {
		logrus.Error("Fake server shutdown error ", err)
		os.Exit(-1)
	}
}

func fakeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "——hi aws ALB, I'm alive ——\n")
}
