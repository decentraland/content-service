package storage

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/decentraland/content-service/metrics"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	log "github.com/sirupsen/logrus"
)

type s3Storage struct {
	Bucket *string
	ACL    *string
	URL    string
	Agent  *metrics.Agent
}

func newS3(bucket, acl, url string, agent *metrics.Agent) *s3Storage {
	sto := new(s3Storage)
	sto.Bucket = aws.String(bucket)
	sto.ACL = aws.String(acl)
	sto.URL = url
	sto.Agent = agent
	return sto
}

func (sto *s3Storage) GetFile(cid string) string {
	u, _ := url.Parse(sto.URL)
	u.Path = path.Join(u.Path, cid)
	url, _ := url.PathUnescape(u.String())
	return url
}

func (sto *s3Storage) SaveFile(filename string, fileDesc io.Reader, contentType string) (string, error) {
	t := time.Now()
	log.Debugf("Uploading file[%s] to s3Storage", filename)
	sess := session.Must(session.NewSession())

	uploader := &s3manager.Uploader{
		S3:                s3.New(sess, aws.NewConfig().WithEndpoint(sto.URL)),
		PartSize:          s3manager.DefaultUploadPartSize,
		Concurrency:       s3manager.DefaultUploadConcurrency,
		LeavePartsOnError: false,
		MaxUploadParts:    s3manager.MaxUploadParts,
	}

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket:      sto.Bucket,
		Key:         aws.String(filename),
		ACL:         sto.ACL,
		Body:        fileDesc,
		ContentType: aws.String(contentType),
	})
	sto.Agent.RecordStorageTime(time.Since(t))
	if err != nil {
		return "", handleS3Error(err)
	}

	return result.Location, nil
}

func (sto *s3Storage) DownloadFile(cid string, filePath string) error {
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
	downloader := &s3manager.Downloader{
		S3:          s3.New(s, aws.NewConfig().WithEndpoint(sto.URL)),
		PartSize:    s3manager.DefaultDownloadPartSize,
		Concurrency: s3manager.DefaultDownloadConcurrency,
	}

	f, err := os.Create(fp)
	if err != nil {
		log.Errorf("Failed to create file %q, %v", fp, err)
		return InternalError{fmt.Sprintf("failed to create file %q, %v", fp, err)}
	}

	n, err := downloader.Download(f, &s3.GetObjectInput{
		Bucket: sto.Bucket,
		Key:    &cid,
	})
	sto.Agent.RecordRetrieveTime(time.Since(t))

	if err != nil {
		return handleS3Error(err)
	}
	sto.Agent.RecordBytesRetrieved(n)
	log.Debugf("CID[%s] found. %d bytes downloaded from s3Storage to %s", cid, n, filePath)

	return nil
}

func (sto *s3Storage) FileSize(cid string) (int64, error) {
	s := session.Must(session.NewSession())
	client := s3.New(s, aws.NewConfig().WithEndpoint(sto.URL))

	hi := &s3.HeadObjectInput{
		Bucket: sto.Bucket,
		Key:    aws.String(cid),
	}

	res, err := client.HeadObject(hi)
	if err != nil {
		return 0, handleS3Error(err)
	}

	return *res.ContentLength, nil
}

func handleS3Error(err error) error {
	log.Error(err.Error())
	switch e := err.(type) {
	case awserr.RequestFailure:
		if e.StatusCode() == http.StatusNotFound {
			return NotFoundError{"file not found"}
		}
		return err
	default:
		return InternalError{err.Error()}
	}
}
