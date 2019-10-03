package content

import (
	"io"

	"github.com/decentraland/content-service/internal/metrics"
)

type Repository interface {
	GetFile(cid string) string
	SaveFile(filename string, fileDesc io.Reader, contentType string) (string, error)
	DownloadFile(cid string, fileName string) error
	FileSize(cid string) (int64, error)
}

type RepoConfig struct {
	Bucket string
	ACL    string
	URL    string
}

func NewStorage(c RepoConfig, agent *metrics.Agent) Repository {
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
