package storage

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

type S3 struct {
	Bucket *string
	ACL    *string
	URL    string
}

func NewS3(bucket, acl, url string) *S3 {
	sto := new(S3)
	sto.Bucket = aws.String(bucket)
	sto.ACL = aws.String(acl)
	sto.URL = url
	return sto
}

func (sto *S3) GetFile(cid string) string {
	u, _ := url.Parse(sto.URL)
	u.Path = path.Join(u.Path, cid)
	url, _ := url.PathUnescape(u.String())
	return url
}

func (sto *S3) SaveFile(filename string, fileDesc io.Reader) (string, error) {
	sess := session.Must(session.NewSession())

	uploader := s3manager.NewUploader(sess)

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: sto.Bucket,
		Key:    aws.String(filename),
		ACL:    sto.ACL,
		Body:   fileDesc,
	})

	if err != nil {
		fmt.Printf("failed to upload file, %v", err)
		return "", err
	}

	return result.Location, nil
}

func (sto *S3) DownloadFile(cid string, filePath string) error {
	dir := filepath.Dir(filePath)
	fp := filepath.Join(dir, filepath.Base(filePath))

	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	s := session.Must(session.NewSession())
	downloader := s3manager.NewDownloader(s)

	f, err := os.Create(fp)
	if err != nil {
		return fmt.Errorf("failed to create file %q, %v", fp, err)
	}

	_, err = downloader.Download(f, &s3.GetObjectInput{
		Bucket: sto.Bucket,
		Key:    &cid,
	})

	if err != nil {
		return handleS3Error(err, cid)
	}

	return nil
}

func handleS3Error(err error, cid string) error {
	switch e := err.(type) {
	case awserr.RequestFailure:
		if e.StatusCode() == http.StatusNotFound {
			return NotFoundError{fmt.Sprintf("Missing file: %s", cid)}
		}
		return err
	default:
		return err
	}
}
