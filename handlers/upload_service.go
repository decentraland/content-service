package handlers

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/decentraland/content-service/data"
	"github.com/fatih/structs"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/decentraland/content-service/storage"
	"github.com/ipsn/go-ipfs/core"
	"github.com/ipsn/go-ipfs/core/coreunix"
)

type UploadRequest struct {
	Metadata      Metadata                `validate:"required"`
	Manifest      *[]FileMetadata         `validate:"required"`
	UploadedFiles map[string]*FileContent `validate:"required"`
	Scene         *scene                  `validate:"required"`
}

type FileContent struct {
	FileName string
	Content  []byte
}

func (f *FileContent) Reader() io.Reader {
	return bytes.NewReader(f.Content)
}

type UploadService interface {
	ProcessUpload(r *UploadRequest) error
}

type UploadServiceImpl struct {
	Storage     storage.Storage
	RedisClient data.RedisClient
	IpfsNode    *core.IpfsNode
	Auth        data.Authorization
}

func NewUploadService(storage storage.Storage, client data.RedisClient, node *core.IpfsNode, auth data.Authorization) *UploadServiceImpl {
	return &UploadServiceImpl{
		Storage:     storage,
		RedisClient: client,
		IpfsNode:    node,
		Auth:        auth,
	}
}

func (us *UploadServiceImpl) ProcessUpload(r *UploadRequest) error {

	err := validateSignature(us.Auth, r.Metadata)
	if err != nil {
		return err
	}

	err = validateKeyAccess(us.Auth, r.Metadata.PubKey, r.Scene.Scene.Parcels)
	if err != nil {
		return err
	}

	consolidatedContent, err := us.consolidateContent(r.UploadedFiles, r.Manifest)
	if err != nil {
		return err
	}

	err = us.validateRootCid(r.Metadata.RootCid, consolidatedContent)
	if err != nil {
		return err
	}

	err = us.processUploadedFiles(r.UploadedFiles, groupFilePathsByCid(r.Manifest), r.Metadata.RootCid)
	if err != nil {
		return err
	}

	err = storeParcelsInformation(r.Metadata.RootCid, r.Scene.Scene.Parcels, us.RedisClient)
	if err != nil {
		return err
	}

	err = us.RedisClient.StoreMetadata(r.Metadata.RootCid, structs.Map(r.Metadata))
	if err != nil {
		return WrapInInternalError(err)
	}
	return nil
}

// Retrieves an error if the signature is invalid, of if the signature does not corresponds to the given key and message
func validateSignature(a data.Authorization, m Metadata) error {
	valid, err := a.IsSignatureValid(m.RootCid, m.Signature, m.PubKey)
	if err != nil {
		return WrapInInternalError(err)
	} else if !valid {
		return NewBadRequestError("Signature is invalid")
	}
	return nil
}

func (us *UploadServiceImpl) consolidateContent(requestFiles map[string]*FileContent, metadata *[]FileMetadata) (map[string]*FileContent, error) {
	consolidatedData := make(map[string]*FileContent)
	for _, m := range *metadata {
		if strings.HasSuffix(m.Name, "/") {
			continue
		}
		var fc *FileContent
		if f, ok := requestFiles[m.Cid]; ok {
			fc = f
		} else {
			c, err := us.Storage.RetrieveFile(m.Cid)
			if err != nil {
				return nil, err
			}
			fc = &FileContent{FileName: m.Name, Content: c}
		}
		err := us.validateCID(fc, m.Cid)
		if err != nil {
			return nil, err
		}
		consolidatedData[m.Cid] = fc
	}
	return consolidatedData, nil
}

// Check if the expectedCID matches the actual CID for a given file
func (us *UploadServiceImpl) validateCID(file *FileContent, expectedCID string) error {
	actualCID, err := coreunix.Add(us.IpfsNode, file.Reader())
	if err != nil {
		return err
	}
	if expectedCID != actualCID {
		return NewBadRequestError(fmt.Sprintf("File[%s] CID does not match expected value: %s", file.FileName, expectedCID))
	}
	return nil
}

// Retrieves an error if the calculated global CID differs from the expected CID
func (us *UploadServiceImpl) validateRootCid(expectedCID string, files map[string]*FileContent) error {
	actualRootCID, err := us.calculateRootCid(expectedCID, files)
	if err != nil {
		return WrapInInternalError(err)
	}

	if expectedCID != actualRootCID {
		return NewBadRequestError("Generated root CID does not match given root CID")
	}
	return nil
}

// Calculate the RootCid for a given set of files
// rootPath: root folder to group the files
// filesMeta: Information about each file path
// files: A map with all the files content
func (us *UploadServiceImpl) calculateRootCid(receivedCid string, files map[string]*FileContent) (string, error) {
	rootDir := filepath.Join("/tmp", receivedCid)

	if err := createTemporaryProjectDir(rootDir, files); err != nil {
		return "", err
	}

	rcid, err := coreunix.AddR(us.IpfsNode, rootDir)
	if err != nil {
		return "", err
	}

	if err := os.RemoveAll(rootDir); err != nil {
		log.Printf("Failed to remove tmp directory: %s", rootDir)
	}

	return rcid, nil
}

func createTemporaryProjectDir(rootDir string, files map[string]*FileContent) error {

	for _, f := range files {
		if f.FileName[len(f.FileName)-1:] == "/" {
			continue
		}

		// This anonymous function would allow the defers to work properly
		// preventing resources from being piled up
		err := func() error {
			dir := filepath.Join(rootDir, filepath.Dir(f.FileName))
			filePath := filepath.Join(dir, filepath.Base(f.FileName))

			err := os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return err
			}

			dst, err := os.Create(filePath)
			if err != nil {
				return err
			}
			defer dst.Close()

			_, err = io.Copy(dst, f.Reader())
			if err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			return err
		}
	}
	return nil
}

// Retrieves an error if the given pKey does not have permissions to modify the parcels
func validateKeyAccess(a data.Authorization, pKey string, parcels []string) error {
	canModify, err := a.UserCanModifyParcels(pKey, parcels)
	if err != nil {
		return WrapInBadRequestError(err)
	} else if !canModify {
		return StatusError{http.StatusUnauthorized, errors.New("address is not authorized to modify given parcels")}
	}
	return nil
}

// Validate and store all the uploaded files
func (us *UploadServiceImpl) processUploadedFiles(fh map[string]*FileContent, paths map[string][]string, cid string) error {
	for fileCID, fileHeader := range fh {

		// This anonymous function would allow the defers to work properly
		// preventing resources from being piled up
		_, err := us.Storage.SaveFile(fileCID, fileHeader.Reader())
		if err != nil {
			return WrapInInternalError(err)
		}

		for _, path := range paths[fileCID] {
			err = us.RedisClient.StoreContent(cid, path, fileCID)
			if err != nil {
				return WrapInInternalError(err)
			}
		}

		if err = us.RedisClient.AddCID(fileCID); err != nil {
			return WrapInInternalError(err)
		}
	}
	return nil
}

func storeParcelsInformation(rootCID string, parcels []string, rc data.RedisClient) error {
	for _, parcel := range parcels {
		err := rc.SetKey(parcel, rootCID)
		if err != nil {
			return WrapInInternalError(err)
		}
	}
	return nil
}
