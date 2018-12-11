package handlers

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/decentraland/content-service/data"
	"github.com/fatih/structs"
	"github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-cid"
	"github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-verifcid"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/decentraland/content-service/storage"
	"github.com/ipsn/go-ipfs/core"
	"github.com/ipsn/go-ipfs/core/coreunix"
)

type UploadRequest struct {
	Metadata      Metadata                           `validate:"required"`
	Manifest      *[]FileMetadata                    `validate:"required"`
	UploadedFiles map[string][]*multipart.FileHeader `validate:"required"`
	Scene         *scene                             `validate:"required"`
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

	err = us.validateContentCID(r.UploadedFiles, r.Manifest, r.Metadata.RootCid)
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

// Retrieves an error if the calculated global CID differs from the expected CID
func (us *UploadServiceImpl) validateContentCID(requestFiles map[string][]*multipart.FileHeader, manifest *[]FileMetadata, rootCid string) error {
	if err := checkCIDFormat(rootCid); err != nil {
		return err
	}

	rootDir := filepath.Join("/tmp", rootCid)
	defer cleanUpTmpFile(rootDir)

	err := us.consolidateContent(requestFiles, manifest, rootDir)
	if err != nil {
		return err
	}

	actualRootCID, err := us.calculateRootCid(rootDir)
	if err != nil {
		return WrapInInternalError(err)
	}

	if rootCid != actualRootCID {
		return NewBadRequestError("Generated root CID does not match given root CID")
	}
	return nil
}

// Consolidate all the scene content under a tmp directory
func (us *UploadServiceImpl) consolidateContent(requestFiles map[string][]*multipart.FileHeader, manifest *[]FileMetadata, projectTmpFile string) error {
	for _, m := range *manifest {
		if strings.HasSuffix(m.Name, "/") {
			continue
		}
		if err := checkCIDFormat(m.Cid); err != nil {
			return err
		}

		tmpFilePath := filepath.Join(projectTmpFile, m.Name)

		var err error
		if f, ok := requestFiles[m.Cid]; ok {
			err = saveRequestFile(f[0], tmpFilePath)
		} else {
			err = us.Storage.DownloadFile(m.Cid, tmpFilePath)
		}
		if err != nil {
			return handleStorageError(err)
		}
		if err := us.validateCID(tmpFilePath, m.Cid); err != nil {
			return err
		}
	}
	return nil
}

func saveRequestFile(f *multipart.FileHeader, projectTmpFile string) error {
	dir := filepath.Dir(projectTmpFile)
	filePath := filepath.Join(dir, filepath.Base(projectTmpFile))

	err := os.MkdirAll(dir, os.ModePerm)
	if err != nil {
		return err
	}

	dst, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	file, err := f.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		return err
	}
	return nil
}

// Check if the expectedCID matches the actual CID for a given file
func (us *UploadServiceImpl) validateCID(f string, expectedCID string) error {
	file, err := os.Open(f)
	if err != nil {
		return NewBadRequestError(fmt.Sprintf("Unable to open File[%s] to calculate CID", f))
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	actualCID, err := coreunix.Add(us.IpfsNode, reader)
	if err != nil {
		return err
	}
	if expectedCID != actualCID {
		return NewBadRequestError(fmt.Sprintf("File[%s] CID does not match expected value: %s", f, expectedCID))
	}
	return nil
}

// Calculate the RootCid for a given set of files
// rootPath: root folder to group the files
func (us *UploadServiceImpl) calculateRootCid(rootPath string) (string, error) {
	rcid, err := coreunix.AddR(us.IpfsNode, rootPath)
	if err != nil {
		return "", err
	}
	return rcid, nil
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

func (us *UploadServiceImpl) processUploadedFiles(fh map[string][]*multipart.FileHeader, paths map[string][]string, cid string) error {
	for fileCID, fileHeaders := range fh {
		fileHeader := fileHeaders[0]

		// This anonymous function would allow the defers to work properly
		// preventing resources from being piled up
		err := func() error {
			file, err := fileHeader.Open()
			if err != nil {
				return WrapInInternalError(err)
			}
			defer file.Close()

			_, err = us.Storage.SaveFile(fileCID, file)
			if err != nil {
				return WrapInInternalError(err)
			}
			return nil
		}()
		if err != nil {
			return err
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

func handleStorageError(err error) error {
	switch e := err.(type) {
	case storage.NotFoundError:
		return WrapInBadRequestError(e)
	default:
		return err
	}
}

// Gruops all the files in the list by file CID
// The map will cointain an entry for each CID, and the associated value would be a list of all the paths
func groupFilePathsByCid(files *[]FileMetadata) map[string][]string {
	filesPaths := make(map[string][]string)
	for _, fileMeta := range *files {
		paths := filesPaths[fileMeta.Cid]
		if paths == nil {
			paths = []string{}
		}
		filesPaths[fileMeta.Cid] = append(paths, fileMeta.Name)
	}
	return filesPaths
}

func cleanUpTmpFile(rootPath string) {
	if _, err := os.Stat(rootPath); err == nil {
		if err := os.RemoveAll(rootPath); err != nil {
			log.Printf("Failed to remove tmp directory: %s", rootPath)
		}
	}
}

func checkCIDFormat(c string) error {
	res, err := cid.Parse(c)
	if err != nil {
		return NewBadRequestError(fmt.Sprintf("Invalid cid: %s", c))
	}
	if err := verifcid.ValidateCid(res); err != nil {
		return NewBadRequestError(fmt.Sprintf("Invalid cid: %s", c))
	}
	return nil
}
