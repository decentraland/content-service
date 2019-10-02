package storage

import (
	"io"

	"github.com/decentraland/content-service/internal/metrics"
)

type Storage interface {
	GetFile(cid string) string
	SaveFile(filename string, fileDesc io.Reader, contentType string) (string, error)
	DownloadFile(cid string, fileName string) error
	FileSize(cid string) (int64, error)
}

type ContentBucket struct {
	Bucket string
	ACL    string
	URL    string
}

func NewStorage(c ContentBucket, agent *metrics.Agent) Storage {
	return newS3(c.Bucket, c.ACL, c.URL, agent)
}

type NotFoundError struct {
	Cause string
}

func (e NotFoundError) Error() string {
	return e.Cause
}

type InternalError struct {
	Cause string
}

func (e InternalError) Error() string {
	return e.Cause
}
