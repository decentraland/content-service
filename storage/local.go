package storage

import (
	"io"
	"os"
	"path/filepath"
)

type Local struct {
	Dir string
}

func NewLocal(dir string) *Local {
	sto := new(Local)
	sto.Dir = dir
	return sto
}

func (sto *Local) CreateLocalDir() error {
	return os.MkdirAll(sto.Dir, os.ModePerm)
}

func (sto *Local) GetFile(cid string) string {
	return sto.Dir + cid
}

func (sto *Local) SaveFile(filename string, fileDesc io.ReadCloser) (string, error) {
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
