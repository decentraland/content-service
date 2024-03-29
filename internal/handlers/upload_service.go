package handlers

import (
	"bufio"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/metrics"
	"github.com/fatih/structs"
	"github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-cid"
	"github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-verifcid"

	"github.com/decentraland/content-service/storage"
	"github.com/decentraland/content-service/utils/rpc"
	"github.com/ipsn/go-ipfs/core"
	"github.com/ipsn/go-ipfs/core/coreunix"
	log "github.com/sirupsen/logrus"
)

type UploadRequest struct {
	Metadata      Metadata                           `validate:"required"`
	Manifest      *[]FileMetadata                    `validate:"required"`
	UploadedFiles map[string][]*multipart.FileHeader `validate:"required"`
	Scene         *scene                             `validate:"required"`
	Origin        string
}

type UploadService interface {
	ProcessUpload(r *UploadRequest) error
}

type UploadServiceImpl struct {
	Storage         storage.Storage
	RedisClient     data.RedisClient
	IpfsNode        *core.IpfsNode
	Auth            data.Authorization
	Agent           *metrics.Agent
	ParcelSizeLimit int64
	Workdir         string
	rpc             *rpc.RPC
	Log             *log.Logger
}

func NewUploadService(storage storage.Storage, client data.RedisClient, node *core.IpfsNode, auth data.Authorization,
	agent *metrics.Agent, parcelSizeLimit int64, workdir string,
	rpc *rpc.RPC, l *log.Logger) *UploadServiceImpl {
	return &UploadServiceImpl{
		Storage:         storage,
		RedisClient:     client,
		IpfsNode:        node,
		Auth:            auth,
		Agent:           agent,
		ParcelSizeLimit: parcelSizeLimit,
		Workdir:         workdir,
		rpc:             rpc,
		Log:             l,
	}
}

func (us *UploadServiceImpl) ProcessUpload(r *UploadRequest) error {
	us.Log.Debug("Processing Upload request")
	logUploadRequest(r, us.Log)

	if err := us.validateSignature(us.Auth, r.Metadata); err != nil {
		return err
	}

	if err := validateKeyAccess(us.Auth, r.Metadata.PubKey, r.Scene.Scene.Parcels, us.Log); err != nil {
		return err
	}

	if err := us.validateRequestSize(r); err != nil {
		return err
	}

	t := time.Now()
	err := us.validateContentCID(r.UploadedFiles, r.Manifest, r.Metadata.RootCid)
	us.Agent.RecordUploadRequestValidationTime(time.Since(t))

	if err != nil {
		return err
	}

	pathsByCid := groupFilePathsByCid(r.Manifest)
	if err := us.processUploadedFiles(r.UploadedFiles, pathsByCid, r.Metadata.RootCid); err != nil {
		return err
	}

	if err := us.storeParcelsInformation(r.Metadata.RootCid, r.Scene.Scene.Parcels); err != nil {
		return err
	}

	if err := us.RedisClient.StoreMetadata(r.Metadata.RootCid, structs.Map(r.Metadata)); err != nil {
		return UnexpectedError{Message: "fail to store metadata", error: err}
	}

	sceneCID := ""
	for _, f := range *r.Manifest {
		if strings.Contains(f.Name, "scene.json") {
			sceneCID = f.Cid
			break
		}
	}
	if err := us.RedisClient.SaveRootCidSceneCid(r.Metadata.RootCid, sceneCID); err != nil {
		return UnexpectedError{Message: "fail to save root cid", error: err} //TODO: we can't recover error from here
	}

	us.Agent.RecordUpload(r.Metadata.RootCid, r.Metadata.PubKey, r.Scene.Scene.Parcels, pathsByCid, r.Origin)

	return nil
}

// Retrieves an error if the signature is invalid, of if the signature does not corresponds to the given key and message
func (us *UploadServiceImpl) validateSignature(a data.Authorization, m Metadata) error {
	us.Log.Debugf("Validating signature: %s", m.Signature)

	// ERC 1654 support https://github.com/ethereum/EIPs/issues/1654
	// We need to validate against a contract address whether this is ok or not?
	if len(m.Signature) > 150 {
		signature := m.Signature
		address := m.PubKey
		msg := fmt.Sprintf("%s.%d", m.Value, m.Timestamp)
		valid, err := us.rpc.ValidateDapperSignature(address, msg, signature)
		if err != nil {
			return err
		}
		if !valid {
			return fmt.Errorf("Signature fails to verify for %s", address)
		}
		return nil
	}
	if !a.IsSignatureValid(fmt.Sprintf("%s.%d", m.RootCid, m.Timestamp), m.Signature, m.PubKey) {
		us.Log.Debugf("Invalid signature[%s] for rootCID[%s] and pubKey[%s]", m.RootCid, m.Signature, m.PubKey)
		return InvalidArgument{"Signature is invalid"}
	}
	return nil
}

// Retrieves an error if the calculated global CID differs from the expected CID
func (us *UploadServiceImpl) validateContentCID(requestFiles map[string][]*multipart.FileHeader, manifest *[]FileMetadata, rootCid string) error {
	us.Log.Debugf("Validating content. RootCID: %s", rootCid)
	if err := checkCIDFormat(rootCid, us.Log); err != nil {
		return err
	}

	rootDir := filepath.Join(us.Workdir, rootCid)
	defer cleanUpTmpFile(rootDir, us.Log)

	us.Log.Infof("Consolidating scene content for CID[%s]", rootCid)
	err := us.consolidateContent(requestFiles, manifest, rootDir)
	if err != nil {
		return err
	}

	actualRootCID, err := us.calculateRootCid(rootDir)
	if err != nil {
		return UnexpectedError{"", err}
	}

	if rootCid != actualRootCID {
		return InvalidArgument{"Generated root CID does not match given root CID"}
	}
	return nil
}

// Consolidate all the scene content under a tmp directory
func (us *UploadServiceImpl) consolidateContent(requestFiles map[string][]*multipart.FileHeader, manifest *[]FileMetadata, projectTmpFile string) error {
	us.Log.Debug("Consolidating Content...")
	for _, m := range *manifest {
		us.Log.Debugf("Verifying Manifest File[%s] CID [%s]", m.Name, m.Cid)
		if strings.HasSuffix(m.Name, "/") {
			continue
		}
		if err := checkCIDFormat(m.Cid, us.Log); err != nil {
			us.Log.Debugf("Invalid CID for fileName[%s] CID [%s]", m.Name, m.Cid)
			return err
		}

		tmpFilePath := filepath.Join(projectTmpFile, m.Name)

		var err error
		if f, ok := requestFiles[m.Cid]; ok {
			err = saveRequestFile(f[0], tmpFilePath, us.Log)
		} else {
			us.Log.Debugf("File[%s] CID [%s] not found in the request content", m.Name, m.Cid)
			err = us.retrieveContent(m.Cid, tmpFilePath)
		}
		if err != nil {
			return err
		}
		if err := us.validateCID(tmpFilePath, m.Cid); err != nil {
			us.Log.Debugf("Failed to validate File[%s] cid: %s", m.Name, err.Error())
			return err
		}
	}
	return nil
}

func saveRequestFile(f *multipart.FileHeader, projectTmpFile string, log *log.Logger) error {
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
	us.Log.Debugf("Validating File[%s] CID, expected: %s", f, expectedCID)
	file, err := os.Open(f)
	if err != nil {
		us.Log.Debugf("Unable to open File[%s] to calculate CID", f)
		return InvalidArgument{fmt.Sprintf("Unable to open File[%s] to calculate CID", f)}
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	actualCID, err := coreunix.Add(us.IpfsNode, reader)
	if err != nil {
		return err
	}
	if expectedCID != actualCID {
		us.Log.Debugf("File[%s] CID does not match expected value: %s", f, expectedCID)
		return InvalidArgument{fmt.Sprintf("File[%s] CID does not match expected value: %s", f, expectedCID)}
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
func validateKeyAccess(a data.Authorization, pKey string, parcels []string, log *log.Logger) error {
	log.Debugf("Validating address: %s", pKey)
	canModify, err := a.UserCanModifyParcels(pKey, parcels)
	if err != nil {
		log.WithError(err).Debugf("Error validating PublicKey[%s]", pKey)
		return InvalidArgument{fmt.Sprintf("Error validating PublicKey[%s]", pKey)}
	} else if !canModify {
		log.Debugf("PublicKey[%s] is not allowed to modify parcels", pKey)
		return UnauthorizedError{"address is not authorized to modify given parcels"}
	}
	return nil
}

func (us *UploadServiceImpl) processUploadedFiles(fh map[string][]*multipart.FileHeader, paths map[string][]string, cid string) error {
	us.Log.Infof("Processing  new content for RootCID[%s]. New files: %d", cid, len(fh))
	for fileCID, fileHeaders := range fh {
		fileHeader := fileHeaders[0]
		us.Log.Debugf("Processing file[%s], CID[%s]", fileHeader.Filename, fileCID)

		// This anonymous function would allow the defers to work properly
		// preventing resources from being piled up
		err := func() error {
			file, err := fileHeader.Open()
			if err != nil {
				us.Log.Errorf("Failed to open file[%s] fileCID[%s]", fileHeader.Filename, fileCID)
				return UnexpectedError{"fail to open file", err}
			}
			defer file.Close()

			_, err = us.Storage.SaveFile(fileCID, file, fileHeader.Header.Get("Content-Type"))
			if err != nil {
				us.Log.Errorf("Failed to store file[%s] fileCID[%s]", fileHeader.Filename, fileCID)
				return UnexpectedError{"fail to store file", err}
			}
			us.Agent.RecordBytesStored(fileHeader.Size)
			us.Log.Infof("File[%s] stored successfully under CID[%s]. Bytes stored: %d", fileHeader.Filename, fileCID, fileHeader.Size)
			return nil
		}()
		if err != nil {
			us.Log.Debugf("Failed to upload file[%s], CID[%s]: %s", fileHeader.Filename, fileCID, err.Error())
			return err
		}
	}

	// Update the content of the parcel with all the files contained in the new scene
	for fileCID, filePaths := range paths {
		for _, p := range filePaths {
			if err := us.RedisClient.StoreContent(cid, p, fileCID); err != nil {
				return UnexpectedError{"redis: fail to store content", err}
			}
		}
		if err := us.RedisClient.AddCID(fileCID); err != nil {
			return UnexpectedError{"redis: fail to store file cid", err}
		}
	}

	us.Log.Infof("[Process New Files] New content for RootCID[%s] done", cid)
	return nil
}

// Retrieves the specify content by the CID from the storage and saves it into the storePath
func (us *UploadServiceImpl) retrieveContent(cid string, storePath string) error {
	err := us.Storage.DownloadFile(cid, storePath)
	if err != nil {
		return handleStorageError(err, cid, us.Log)
	}

	return nil
}

func (us *UploadServiceImpl) storeParcelsInformation(rootCID string, parcels []string) error {

	err := us.RedisClient.SetSceneParcels(rootCID, parcels)
	if err != nil {
		us.Log.WithError(err).Errorf("Error when storing parcels for root cid %s", rootCID)
		return UnexpectedError{"redis: fail to store parcel cid", err}
	}

	for _, parcel := range parcels {

		err = us.RedisClient.SetProcessedParcel(parcel)
		if err != nil {
			us.Log.WithError(err).Errorf("Unable to store parcel[%s] ", parcel)
			return UnexpectedError{"redis: fail to store parcel information", err}
		}
	}

	return err
}

func (us *UploadServiceImpl) validateRequestSize(r *UploadRequest) error {
	maxSize := int64(len(r.Scene.Scene.Parcels)) * us.ParcelSizeLimit

	size, err := us.estimateRequestSize(r)
	if err != nil {
		return err
	}

	if size > maxSize {
		us.Log.Errorf(fmt.Sprintf("UploadRequest RootCid[%s] exceeds the allowed limit Max[bytes]: %d, RequestSize[bytes]: %d", r.Metadata.RootCid, maxSize, size))
		return InvalidArgument{fmt.Sprintf("UploadRequest exceeds the allowed limit Max[bytes]: %d, RequestSize[bytes]: %d", maxSize, size)}
	}
	return nil
}

func (us *UploadServiceImpl) estimateRequestSize(r *UploadRequest) (int64, error) {
	size := int64(0)
	for _, m := range *r.Manifest {
		if strings.HasSuffix(m.Name, "/") {
			continue
		}
		if f, ok := r.UploadedFiles[m.Cid]; ok {
			size += f[0].Size
		} else {
			s, err := us.retrieveUploadedFileSize(m.Cid)
			if err != nil {
				return 0, err
			}
			size += s
		}
	}
	us.Log.Debugf(fmt.Sprintf("UploadRequest size: %d", size))
	return size, nil
}

func (us *UploadServiceImpl) retrieveUploadedFileSize(cid string) (int64, error) {
	size, err := us.Storage.FileSize(cid)
	if err != nil {
		return 0, handleStorageError(err, cid, us.Log)
	}
	return size, nil
}

func handleStorageError(err error, cid string, log *log.Logger) error {
	switch e := err.(type) {
	case storage.NotFoundError:
		log.Debugf("file with cid[%s] not found", cid)
		return InvalidArgument{fmt.Sprintf("file: %s not found", cid)}
	default:
		log.WithError(e).Error("Storage Error")
		return UnexpectedError{"storage error", err}
	}
}

// Groups all the files in the list by file CID
// The map will contain an entry for each CID, and the associated value would be a list of all the paths
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

func cleanUpTmpFile(rootPath string, log *log.Logger) {
	if _, err := os.Stat(rootPath); err == nil {
		if err := os.RemoveAll(rootPath); err != nil {
			log.WithError(err).Errorf("Failed to remove tmp directory: %s", rootPath)
		}
	}
}

func checkCIDFormat(c string, log *log.Logger) error {
	res, err := cid.Parse(c)
	if err != nil {
		log.Debugf("Invalid cid: %s", c)
		return InvalidArgument{fmt.Sprintf("invalid cid: %s", c)}
	}
	if err := verifcid.ValidateCid(res); err != nil {
		log.Debugf("Invalid cid: %s", c)
		return InvalidArgument{fmt.Sprintf("invalid cid: %s", c)}
	}
	return nil
}

func logUploadRequest(r *UploadRequest, l *log.Logger) {
	var md []string
	for _, m := range *r.Manifest {
		md = append(md, fmt.Sprintf("%s[%s]", m.Name, m.Cid))
	}
	var rd []string
	for _, v := range r.UploadedFiles {
		h := v[0]
		rd = append(rd, fmt.Sprintf("%s[%d bytes]", h.Filename, h.Size))
	}

	l.WithFields(log.Fields{
		"parcel":       r.Scene.Main,
		"requestFiles": strings.Join(rd, ", "),
		"manifest":     strings.Join(md, ", "),
		"key":          r.Metadata.PubKey,
		"signature":    r.Metadata.Signature,
	}).Info("Incoming upload request")
}
