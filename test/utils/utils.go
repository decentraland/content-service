package utils

import (
	"bufio"
	"context"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ipsn/go-ipfs/core"
	"github.com/ipsn/go-ipfs/core/coreunix"
)

type FileMetadata struct {
	Cid  string `json:"cid" validate:"required"`
	Name string `json:"name" validate:"required"`
}

func ToFileData(workDir string, node *core.IpfsNode) ([]*FileMetadata, error) {
	var result []*FileMetadata
	err := filepath.Walk(workDir,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.Mode().IsDir() {
				cid, err := CalculateFileCID(path, node)
				if err != nil {
					return err
				}
				result = append(result, &FileMetadata{
					Name: strings.Replace(path, workDir, "", -1),
					Cid:  cid,
				})
			}
			return nil
		})
	if err != nil {
		log.Println(err)
	}
	return result, nil
}

func InitIpfsNode() (*core.IpfsNode, error) {
	ctx, _ := context.WithCancel(context.Background())
	return core.NewNode(ctx, nil)
}

// Calculates a file CID
func CalculateFileCID(f string, node *core.IpfsNode) (string, error) {
	file, err := os.Open(f)
	if err != nil {
		return "", err
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	actualCID, err := coreunix.Add(node, reader)
	if err != nil {
		return "", err
	}

	return actualCID, nil
}

// Calculate the RootCid for a given set of files
// rootPath: root folder to group the files
func CalculateRootCid(rootPath string, node *core.IpfsNode) (string, error) {
	rcid, err := coreunix.AddR(node, rootPath)
	if err != nil {
		return "", err
	}
	return rcid, nil
}
