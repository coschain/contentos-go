package commands

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/coschain/cobra"
	"github.com/sirupsen/logrus"
)

var dataDir string
var interval int32
//var destAddr string

const (
	S3_REGION    = "eu-north-1" // lightsail.RegionNameEuCentral1
	S3_BUCKET    = "cosd-databackup"
	archFileName = "data.tar.gz"
)

var AgentCmd = func() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent",
		Short: "start backup agent",
		Long:  "start back agent and regularly sends cosd data files to backup server",
		Run:   startBackUpAgent,
	}
	cmd.Flags().StringVarP(&dataDir, "data_dir", "d", "", "directory of cosd data")
	cmd.Flags().Int32VarP(&interval, "interval", "i", 86400, "backup data every interval seconds")
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
	cmd := exec.Command("/bin/bash","-c", "pkill cosd")
	if err := cmd.Run(); err != nil {
		logrus.Error(err)
	}
	if err := zip(); err == nil {
		logrus.Info("cosd data archived at ", time.Now())
	} else {
		logrus.Error(err)
		return err
	}

	//exec.Command("/bin/bash","-c","/data/coschain/contentos-go/cmd/cosd/cosd start -n /data/logs/coschain/cosd/")
	cmd = exec.Command("/bin/bash","-c","/data/coschain/src/deploy/boot.sh")
	if _, err := cmd.Output(); err != nil {
		logrus.Error(err)
		return err
	}

	if err := SendToS3(); err == nil {
		logrus.Info("cosd data uploaded at ", time.Now())
	} else {
		logrus.Error(err)
		return err
	}
	os.Remove(archFileName)

	return nil
}

func zip() error {
	input := make([]*os.File, 3)
	inputName := []string{
		dataDir + "/blog",
		dataDir + "/checkpoint",
		dataDir + "/db",
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

/*
func Upload(url, file string) (err error) {
	// Prepare a form that you will submit to that URL.
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	// Add your image file
	f, err := os.Open(file)
	if err != nil {
		return
	}
	defer f.Close()
	fw, err := w.CreateFormFile("file", file)
	if err != nil {
		return
	}
	if _, err = io.Copy(fw, f); err != nil {
		return
	}

	// Add the other fields
	if fw, err = w.CreateFormField("key"); err != nil {
		return
	}
	if _, err = fw.Write([]byte("KEY")); err != nil {
		return
	}
	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Submit the request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", res.Status)
	}
	return
}

func UploadTwo() (err error) {
	url := "http://localhost:8062/many"
	// Prepare a form that you will submit to that URL.
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	for i := 1; i <= 100; i++ {
		fname := fmt.Sprintf("file%d.bin", i)
		fw, err2 := w.CreateFormFile("file", fname)
		if err2 != nil {
			err = err2
			return
		}

		f, err2 := os.Open(fname)
		if err2 != nil {
			err = err2
			return
		}
		if _, err2 = io.Copy(fw, f); err2 != nil {
			err = err2
			return
		}
		f.Close()
	}

	// Add the other fields
	fw, err2 := w.CreateFormField("key")
	if err2 != nil {
		err = err2
		return
	}
	if _, err2 = fw.Write([]byte("KEY")); err2 != nil {
		err = err2
		return
	}

	// Don't forget to close the multipart writer.
	// If you don't close it, your request will be missing the terminating boundary.
	w.Close()

	// Now that you have a form, you can submit it to your handler.
	req, err := http.NewRequest("POST", url, &b)
	if err != nil {
		return
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", w.FormDataContentType())

	// Submit the request
	client := &http.Client{}
	res, err := client.Do(req)
	if err != nil {
		return
	}

	// Check the response
	if res.StatusCode != http.StatusOK {
		err = fmt.Errorf("bad status: %s", res.Status)
	}
	return
}
*/

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
			logrus.Warnf("skip file %s coz it's irregular")
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

	// Upload
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
	now := time.Now().String()
	_, err = s3.New(s).PutObject(&s3.PutObjectInput{
		Bucket:               aws.String(S3_BUCKET),
		Key:                  aws.String(now+"/"+fileDir),
		//ACL:                  aws.String("private"),
		Body:                 file,//bytes.NewReader(buffer),
		//ContentLength:        aws.Int64(size),
		//ContentType:          aws.String(http.DetectContentType(buffer)),
		//ContentDisposition:   aws.String("attachment"),
		//ServerSideEncryption: aws.String("AES256"),
	})
	return err
}
