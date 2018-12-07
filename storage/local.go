package storage

import (
	"fmt"
	"io"
	"io/ioutil"
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

func (sto *Local) SaveFile(filename string, fileDesc io.Reader) (string, error) {
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

func (sto *Local) RetrieveFile(cid string) ([]byte, error) {
	path := filepath.Join(sto.Dir, cid)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, &NotFoundError{fmt.Sprintf("Missing file: %s", cid)}
	}

	content, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return content, nil
}
