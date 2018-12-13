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
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/decentraland/content-service/storage"
	"github.com/ipsn/go-ipfs/core"
	"github.com/ipsn/go-ipfs/core/coreunix"
	log "github.com/sirupsen/logrus"
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
	logUploadRequest(r)

	if err := validateSignature(us.Auth, r.Metadata); err != nil {
		return err
	}

	if err := validateKeyAccess(us.Auth, r.Metadata.PubKey, r.Scene.Scene.Parcels); err != nil {
		return err
	}

	if err := us.validateContentCID(r.UploadedFiles, r.Manifest, r.Metadata.RootCid); err != nil {
		return err
	}

	if err := us.processUploadedFiles(r.UploadedFiles, groupFilePathsByCid(r.Manifest), r.Metadata.RootCid); err != nil {
		return err
	}

	if err := storeParcelsInformation(r.Metadata.RootCid, r.Scene.Scene.Parcels, us.RedisClient); err != nil {
		return err
	}

	if err := us.RedisClient.StoreMetadata(r.Metadata.RootCid, structs.Map(r.Metadata)); err != nil {
		return WrapInInternalError(err)
	}
	return nil
}

// Retrieves an error if the signature is invalid, of if the signature does not corresponds to the given key and message
func validateSignature(a data.Authorization, m Metadata) error {
	if !a.IsSignatureValid(m.RootCid, m.Signature, m.PubKey) {
		log.Debugf("Invalid signature[%s] for rootCID[%s] and pubKey[%s]", m.RootCid, m.Signature, m.PubKey)
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

	log.Infof("Consolidating scene content for CID[%s]", rootCid)
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
		log.Debugf("Verifying Manifest File[%s] CID [%s]", m.Name, m.Cid)
		if strings.HasSuffix(m.Name, "/") {
			continue
		}
		if err := checkCIDFormat(m.Cid); err != nil {
			log.Debugf("Invalid CID for fileName[%s] CID [%s]", m.Name, m.Cid)
			return err
		}

		tmpFilePath := filepath.Join(projectTmpFile, m.Name)

		var err error
		if f, ok := requestFiles[m.Cid]; ok {
			err = saveRequestFile(f[0], tmpFilePath)
		} else {
			log.Debugf("File[%s] CID [%s] not found in the request content", m.Name, m.Cid)
			err = us.retrieveContent(m.Cid, tmpFilePath)
		}
		if err != nil {
			return err
		}
		if err := us.validateCID(tmpFilePath, m.Cid); err != nil {
			log.Debugf("Failed to validate File[%s] cid: %s", m.Name, err.Error())
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
		log.Errorf("Failed to create directory: %s", dir)
		return err
	}

	dst, err := os.Create(filePath)
	if err != nil {
		log.Errorf("Failed to create file: %s", filePath)
		return err
	}
	defer dst.Close()

	file, err := f.Open()
	if err != nil {
		log.Errorf("Failed to Open file: %s", filePath)
		return err
	}
	defer file.Close()

	_, err = io.Copy(dst, file)
	if err != nil {
		log.Errorf("Failed to save file: %s", filePath)
		return err
	}
	return nil
}

// Check if the expectedCID matches the actual CID for a given file
func (us *UploadServiceImpl) validateCID(f string, expectedCID string) error {
	log.Debugf("Validating File[%s] CID, expected: %s", f, expectedCID)
	file, err := os.Open(f)
	if err != nil {
		log.Debugf("Unable to open File[%s] to calculate CID", f)
		return NewBadRequestError(fmt.Sprintf("Unable to open File[%s] to calculate CID", f))
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	actualCID, err := coreunix.Add(us.IpfsNode, reader)
	if err != nil {
		return err
	}
	if expectedCID != actualCID {
		log.Debugf("File[%s] CID does not match expected value: %s", f, expectedCID)
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
		log.Infof("Error validating PublicKey[%s]", pKey)
		return WrapInBadRequestError(err)
	} else if !canModify {
		log.Infof("PublicKey[%s] is not allow to modify parcels", pKey)
		return StatusError{http.StatusUnauthorized, errors.New("address is not authorized to modify given parcels")}
	}
	return nil
}

func (us *UploadServiceImpl) processUploadedFiles(fh map[string][]*multipart.FileHeader, paths map[string][]string, cid string) error {
	log.Infof("Processing  new content for RootCID[%s]. New files: %d", cid, len(fh))
	for fileCID, fileHeaders := range fh {
		fileHeader := fileHeaders[0]
		log.Debugf("Processing file[%s], CID[%s]", fileHeader.Filename, fileCID)

		// This anonymous function would allow the defers to work properly
		// preventing resources from being piled up
		err := func() error {
			file, err := fileHeader.Open()
			if err != nil {
				log.Errorf("Failed to open file[%s] fileCID[%s]", fileHeader.Filename, fileCID)
				return WrapInInternalError(err)
			}
			defer file.Close()

			_, err = us.Storage.SaveFile(fileCID, file)
			if err != nil {
				log.Errorf("Failed to store file[%s] fileCID[%s]", fileHeader.Filename, fileCID)
				return WrapInInternalError(err)
			}
			log.Infof("File[%s] stored successfully under CID[%s]. Bytes stored: %d", fileHeader.Filename, fileCID, fileHeader.Size)
			return nil
		}()
		if err != nil {
			log.Debugf("Failed to upload file[%s], CID[%s]: %s", fileHeader.Filename, fileCID, err.Error())
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
	log.Infof("[Process New Files] New content for RootCID[%s] done", cid)
	return nil
}

// Retrieves the specify content by the CID from the storage and saves it into the storePath
func (us *UploadServiceImpl) retrieveContent(cid string, storePath string) error {
	stored, err := us.RedisClient.IsContentMember(cid)
	if err != nil {
		log.Errorf("Failed to verify content, CID[%s]: %s", cid, err.Error())
		return NewInternalError(fmt.Sprintf("Failed to retrieve content CID[%s]", cid))
	}

	if !stored {
		log.Debugf("CID[%s] not found in storage and was not provided in the request", cid)
		return NewBadRequestError(fmt.Sprintf("CID[%s] not found in storage and was not provided in the request", cid))
	}

	err = us.Storage.DownloadFile(cid, storePath)

	if err != nil {
		return handleStorageError(err)
	}

	return nil
}

func storeParcelsInformation(rootCID string, parcels []string, rc data.RedisClient) error {
	for _, parcel := range parcels {
		err := rc.SetKey(parcel, rootCID)
		if err != nil {
			log.Errorf("Unable to store parcel[%s] Information: %s ", parcel, err.Error())
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
		return NewInternalError("Failed to store request content")
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
			log.Errorf("Failed to remove tmp directory: %s", rootPath)
		}
	}
}

func checkCIDFormat(c string) error {
	res, err := cid.Parse(c)
	if err != nil {
		log.Debugf("Invalid cid: %s", c)
		return NewBadRequestError(fmt.Sprintf("Invalid cid: %s", c))
	}
	if err := verifcid.ValidateCid(res); err != nil {
		log.Debugf("Invalid cid: %s", c)
		return NewBadRequestError(fmt.Sprintf("Invalid cid: %s", c))
	}
	return nil
}

func logUploadRequest(r *UploadRequest) {
	var md []string
	for _, m := range *r.Manifest {
		md = append(md, fmt.Sprintf("%s[%s]", m.Name, m.Cid))
	}
	var rd []string
	for _, v := range r.UploadedFiles {
		h := v[0]
		rd = append(rd, fmt.Sprintf("%s[%d bytes]", h.Filename, h.Size))
	}

	log.WithFields(log.Fields{
		"parcel":       r.Scene.Main,
		"requestFiles": strings.Join(rd, ", "),
		"manifest":     strings.Join(md, ", "),
		"key":          r.Metadata.PubKey,
		"signature":    r.Metadata.Signature,
	}).Info("Incoming upload request")
}