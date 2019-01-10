package storage

import (
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/decentraland/content-service/metrics"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/sirupsen/logrus"
)

type S3 struct {
	Bucket *string
	ACL    *string
	URL    string
	Agent  metrics.Agent
}

func NewS3(bucket, acl, url string, agent metrics.Agent) *S3 {
	sto := new(S3)
	sto.Bucket = aws.String(bucket)
	sto.ACL = aws.String(acl)
	sto.URL = url
	sto.Agent = agent
	return sto
}

func (sto *S3) GetFile(cid string) string {
	u, _ := url.Parse(sto.URL)
	u.Path = path.Join(u.Path, cid)
	url, _ := url.PathUnescape(u.String())
	return url
}

func (sto *S3) SaveFile(filename string, fileDesc io.Reader) (string, error) {
	t := time.Now()
	log.Debugf("Uploading file[%s] to S3", filename)
	sess := session.Must(session.NewSession())

	uploader := s3manager.NewUploader(sess)

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: sto.Bucket,
		Key:    aws.String(filename),
		ACL:    sto.ACL,
		Body:   fileDesc,
	})
	sto.Agent.RecordStorageTime(time.Since(t))
	if err != nil {
		log.Errorf("Fail to upload file: %s", err.Error())
		return "", err
	}

	return result.Location, nil
}

func (sto *S3) DownloadFile(cid string, filePath string) error {
	t := time.Now()
	log.Debugf("Downloading Key[%s] to File[%s]", cid, filePath)
	dir := filepath.Dir(filePath)
	fp := filepath.Join(dir, filepath.Base(filePath))

	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		log.Errorf("Unable to generate path: %s", dir)
		return err
	}

	s := session.Must(session.NewSession())
	downloader := s3manager.NewDownloader(s)

	f, err := os.Create(fp)
	if err != nil {
		log.Errorf("Failed to create file %q, %v", fp, err)
		return fmt.Errorf("failed to create file %q, %v", fp, err)
	}

	n, err := downloader.Download(f, &s3.GetObjectInput{
		Bucket: sto.Bucket,
		Key:    &cid,
	})
	sto.Agent.RecordRetrieveTime(time.Since(t))

	if err != nil {
		return handleS3Error(err, cid)
	}
	sto.Agent.RecordBytesRetrieved(n)
	log.Debugf("CID[%s] found. %d bytes downloaded from S3 to %s", cid, n, filePath)

	return nil
}

func handleS3Error(err error, cid string) error {
	switch e := err.(type) {
	case awserr.RequestFailure:
		if e.StatusCode() == http.StatusNotFound {
			log.Debugf("CID[%s] Not Found in S3", cid)
			return NotFoundError{fmt.Sprintf("Missing file: %s", cid)}
		}
		return err
	default:
		log.Error(err.Error())
		return errors.New("An error occurred while accessing content Storage")
	}
}
