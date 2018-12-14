package storage

import (
	"fmt"
	"io"
	"log"
	"strings"

	"github.com/decentraland/content-service/config"
)

type Storage interface {
	GetFile(cid string) string
	SaveFile(filename string, fileDesc io.Reader) (string, error)
	DownloadFile(cid string, fileName string) error
}

func NewStorage(conf *config.Storage) Storage {
	switch config.StorageType(strings.ToUpper(conf.StorageType)) {
	case config.LOCAL:
		return buildLocalStorage(conf)
	case config.REMOTE:
		return NewS3(conf.RemoteConfig.Bucket, conf.RemoteConfig.ACL, conf.RemoteConfig.URL)
	default:
		log.Fatal(fmt.Sprintf("Invalid StorageType: [%s]. Alowed Values: [REMOTE, LOCAL]", conf.StorageType))
	}
	return nil
}

func buildLocalStorage(conf *config.Storage) Storage {
	sto := NewLocal(conf.LocalPath)
	err := sto.CreateLocalDir()
	if err != nil {
		log.Fatal(err)
	}
	return sto
}

type NotFoundError struct {
	Cause string
}

func (e NotFoundError) Error() string {
	return e.Cause
}
