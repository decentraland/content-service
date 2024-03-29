package storage

import (
	"fmt"
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

func (sto *Local) SaveFile(filename string, fileDesc io.Reader, contentType string) (string, error) {
	path := filepath.Join(sto.Dir, filename)
	dst, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	_, err = io.Copy(dst, fileDesc)
	if err != nil {
		return "", err
	}

	return path, nil
}

func (sto *Local) DownloadFile(cid string, fileName string) error {
	path := filepath.Join(sto.Dir, cid)
	in, err := os.Open(path)
	if err != nil {
		return NotFoundError{fmt.Sprintf("Not found: %s", cid)}
	}
	defer in.Close()

	dir := filepath.Dir(fileName)
	fp := filepath.Join(dir, filepath.Base(fileName))

	err = os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	out, err := os.Create(fp)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}
	return out.Close()
}

func (sto *Local) FileSize(cid string) (int64, error) {
	path := filepath.Join(sto.Dir, cid)
	i, err := os.Stat(path)

	if err != nil && os.IsNotExist(err) {
		return 0, NotFoundError{fmt.Sprintf("Not found: %s", cid)}
	} else if err != nil {
		return 0, err
	}

	return i.Size(), nil
}
