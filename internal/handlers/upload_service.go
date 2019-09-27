package handlers

import (
	"bufio"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/decentraland/content-service/internal/ipfs"

	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/metrics"
	"github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-cid"
	"github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-verifcid"

	"github.com/decentraland/content-service/storage"
	"github.com/decentraland/content-service/utils/rpc"
	log "github.com/sirupsen/logrus"
)

type UploadRequest struct {
	Metadata      *Metadata                          `validate:"required"`
	Mappings      []ContentMapping                   `validate:"required"`
	UploadedFiles map[string][]*multipart.FileHeader `validate:"required"`
	Parcels       []string                           `validate:"required"`
	Origin        string
}

// Groups all the files in the list by file CID
// The map will contain an entry for each CID, and the associated value would be a list of all the paths
func (r *UploadRequest) GroupFilePathsByCid() map[string][]string {
	filesPaths := make(map[string][]string)
	for _, fileMeta := range r.Mappings {
		paths := filesPaths[fileMeta.Cid]
		if paths == nil {
			paths = []string{}
		}
		filesPaths[fileMeta.Cid] = append(paths, fileMeta.Name)
	}
	return filesPaths
}

type UploadService interface {
	ProcessUpload(r *UploadRequest) error
}

type UploadServiceImpl struct {
	Storage         storage.Storage
	IpfsHelper      *ipfs.IpfsHelper
	Auth            data.Authorization
	Agent           *metrics.Agent
	ParcelSizeLimit int64
	rpc             *rpc.RPC
	Log             *log.Logger
}

func NewUploadService(storage storage.Storage, helper *ipfs.IpfsHelper, auth data.Authorization,
	agent *metrics.Agent, parcelSizeLimit int64,
	rpc *rpc.RPC, l *log.Logger) *UploadServiceImpl {
	return &UploadServiceImpl{
		Storage:         storage,
		IpfsHelper:      helper,
		Auth:            auth,
		Agent:           agent,
		ParcelSizeLimit: parcelSizeLimit,
		rpc:             rpc,
		Log:             l,
	}
}

func (us *UploadServiceImpl) ProcessUpload(r *UploadRequest) error {
	us.Log.Debug("Processing Upload request")

	if err := us.validateSignature(us.Auth, r.Metadata); err != nil {
		return err
	}

	if err := validateKeyAccess(us.Auth, r.Metadata.PubKey, r.Parcels, us.Log); err != nil {
		return err
	}

	if err := us.validateRequestSize(r); err != nil {
		return err
	}

	t := time.Now()
	err := us.validateRequestContent(r.UploadedFiles, r.Mappings, r.Metadata.SceneCid)
	us.Agent.RecordUploadRequestValidationTime(time.Since(t))

	if err != nil {
		return err
	}

	pathsByCid := r.GroupFilePathsByCid()
	if err := us.processUploadedFiles(r.UploadedFiles, r.Metadata.SceneCid); err != nil {
		return err
	}

	// TODO: Update bucket for all coords. update local cache

	// TODO: Store scene upload metadata

	us.Agent.RecordUpload(r.Metadata.SceneCid, r.Metadata.PubKey, r.Parcels, pathsByCid, r.Origin)

	return nil
}

// Retrieves an error if the signature is invalid, of if the signature does not corresponds to the given key and message
func (us *UploadServiceImpl) validateSignature(a data.Authorization, m *Metadata) error {
	us.Log.Debugf("Validating signature: %s", m.Signature)

	// ERC 1654 support https://github.com/ethereum/EIPs/issues/1654
	// We need to validate against a contract address whether this is ok or not?
	if len(m.Signature) > 150 {
		signature := m.Signature
		address := m.PubKey
		msg := fmt.Sprintf("%s.%d", m.SceneCid, m.Timestamp)
		valid, err := us.rpc.ValidateDapperSignature(address, msg, signature)
		if err != nil {
			return err
		}
		if !valid {
			return fmt.Errorf("signature fails to verify for %s", address)
		}
		return nil
	}
	if !a.IsSignatureValid(fmt.Sprintf("%s.%d", m.SceneCid, m.Timestamp), m.Signature, m.PubKey) {
		us.Log.Debugf("Invalid signature[%s] for SceneCid[%s] and pubKey[%s]", m.SceneCid, m.Signature, m.PubKey)
		return InvalidArgument{"Signature is invalid"}
	}
	return nil
}

// Retrieves an error if the calculated global CID differs from the expected CID
func (us *UploadServiceImpl) validateRequestContent(requestFiles map[string][]*multipart.FileHeader,
	manifest []ContentMapping, sceneCID string) error {

	us.Log.Debugf("Validating content. SceneCID: %s", sceneCID)
	if err := checkCIDFormat(sceneCID, us.Log); err != nil {
		return err
	}
	for _, m := range manifest {
		if strings.HasSuffix(m.Name, "/") {
			continue
		}

		rFile, ok := requestFiles[m.Cid]
		if ok {
			fileCID, err := us.calculateCID(rFile[0], m.Cid)
			if err != nil {
				us.Log.Debugf("Failed to validate File[%s] cid: %s", m.Name, err.Error())
				return err
			}

			if rFile[0].Filename == "scene.json" && fileCID != sceneCID {
				return InvalidArgument{"scene.json cid does not match"}
			}

			if fileCID != m.Cid {
				return InvalidArgument{
					fmt.Sprintf("File[%s] informed CID[%s] does not match: %s", m.Name, m.Cid, fileCID),
				}
			}
		} else {
			s, err := us.Storage.FileSize(m.Cid)
			if err != nil || s <= 0 {
				us.Log.WithError(err).Debugf("File included in the metadata section does not exist Cid: %s", m.Cid)
				return err
			}
		}

	}
	return nil
}

func (us *UploadServiceImpl) calculateCID(file *multipart.FileHeader, expectedCID string) (string, error) {
	us.Log.Debugf("Validating File, expectedCID: %s", expectedCID)
	f, err := file.Open()
	if err != nil {
		us.Log.Debugf("Unable to open File[%s] to calculate CID", f)
		return "", InvalidArgument{fmt.Sprintf("Unable to open File[%s] to calculate CID", f)}
	}
	defer f.Close()

	actualCID, err := us.IpfsHelper.CalculateCID(bufio.NewReader(f))
	if err != nil {
		return "", err
	}
	return actualCID, nil
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

func (us *UploadServiceImpl) processUploadedFiles(fh map[string][]*multipart.FileHeader, cid string) error {
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
	us.Log.Infof("[Process New Files] New content for RootCID[%s] done", cid)
	return nil
}

func (us *UploadServiceImpl) validateRequestSize(r *UploadRequest) error {
	maxSize := int64(len(r.Parcels)) * us.ParcelSizeLimit

	size, err := us.estimateRequestSize(r)
	if err != nil {
		return err
	}

	if size > maxSize {
		us.Log.Errorf(fmt.Sprintf("UploadRequest RootCid[%s] exceeds the allowed limit Max[bytes]: %d, RequestSize[bytes]: %d", r.Metadata.SceneCid, maxSize, size))
		return InvalidArgument{fmt.Sprintf("UploadRequest exceeds the allowed limit Max[bytes]: %d, RequestSize[bytes]: %d", maxSize, size)}
	}
	return nil
}

func (us *UploadServiceImpl) estimateRequestSize(r *UploadRequest) (int64, error) {
	size := int64(0)
	for _, m := range r.Mappings {
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
