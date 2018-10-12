package storage

import (
	"mime/multipart"
)

type Storage interface {
	GetFile(cid string) string
	SaveFile(filename string, fileDesc multipart.File) (string, error)
}
