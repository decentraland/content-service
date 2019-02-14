package storage

import (
	"github.com/decentraland/content-service/metrics"
	"io"
	"strings"

	"github.com/decentraland/content-service/config"
	log "github.com/sirupsen/logrus"
)

type Storage interface {
	GetFile(cid string) string
	SaveFile(filename string, fileDesc io.Reader, contentType string) (string, error)
	DownloadFile(cid string, fileName string) error
	FileSize(cid string) (int64, error)
}

func NewStorage(conf *config.Storage, agent *metrics.Agent) Storage {
	log.Infof("Storage mode: %s", conf.StorageType)
	switch config.StorageType(strings.ToUpper(conf.StorageType)) {
	case config.LOCAL:
		return buildLocalStorage(conf)
	case config.REMOTE:
		return NewS3(conf.RemoteConfig.Bucket, conf.RemoteConfig.ACL, conf.RemoteConfig.URL, agent)
	default:
		log.Fatalf("Invalid Storage Type: %s", conf.StorageType)
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
