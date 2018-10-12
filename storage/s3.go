package storage

import (
	"fmt"
	"mime/multipart"

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
	return sto.URL + cid
}

func (sto *S3) SaveFile(filename string, fileDesc multipart.File) (string, error) {
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
