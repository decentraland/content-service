package handlers

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func saveFile(fileDescriptor multipart.File, localStorageDir string, filename string) (string, error) {
	err := os.MkdirAll(localStorageDir, os.ModePerm)
	if err != nil {
		return "", err
	}

	localPath := filepath.Join(localStorageDir, filename)
	dst, err2 := os.Create(localPath)
	if err2 != nil {
		return "", err2
	}

	_, err = io.Copy(dst, fileDescriptor)
	if err != nil {
		return "", err
	}

	return filename, nil
}

func getFileS3(cid string) string {
	return "https://content-service.s3.amazonaws.com/" + cid
}

func saveFileS3(fileDescriptor multipart.File, filename string) (string, error) {
	sess := session.Must(session.NewSession())

	uploader := s3manager.NewUploader(sess)

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("content-service"),
		Key:    aws.String(filename),
		ACL:    aws.String("public-read"),
		Body:   fileDescriptor,
	})

	if err != nil {
		fmt.Printf("failed to upload file, %v", err)
		return "", err
	}

	return result.Location, nil
}
