package storage

import (
	"io"
	"log"

	"github.com/decentraland/content-service/config"
)

type Storage interface {
	GetFile(cid string) string
	SaveFile(filename string, fileDesc io.ReadCloser) (string, error)
}

func NewStorage(config *config.Configuration) Storage {
	if config.S3Storage.Bucket != "" {
		return NewS3(config.S3Storage.Bucket, config.S3Storage.ACL, config.S3Storage.URL)
	} else {
		sto := NewLocal(config.LocalStorage)
		err := sto.CreateLocalDir()
		if err != nil {
			log.Fatal(err)
		}
		return sto
	}
}
