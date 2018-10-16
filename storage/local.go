package storage

import (
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

type Local struct {
	Dir string
}

func NewLocal(dir string) (*Local, error) {
	sto := new(Local)
	sto.Dir = dir
	return sto, err
}

func (sto *Local) GetFile(cid string) string {
	return sto.Dir + cid
}

func (sto *Local) SaveFile(filename string, fileDesc multipart.File) (string, error) {
	path := filepath.Join(sto.Dir, filename)
	dst, err := os.Create(path)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(dst, fileDesc)
	if err != nil {
		return "", err
	}

	return path, nil
}
