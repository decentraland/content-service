package main

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
)

func getFile(cid string) string {
	return localStorageDir + cid
}

func saveFile(fileDescriptor multipart.File, filename string) (string, error) {
	dst, err := os.Create(localStorageDir + filename)
	if err != nil {
		return "", err
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