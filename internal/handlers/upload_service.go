package handlers

import (
	"bufio"
	"fmt"
	"mime/multipart"
	"strings"
	"time"

	"github.com/decentraland/content-service/internal/deployment"
	"github.com/decentraland/content-service/internal/entities"

	"github.com/decentraland/content-service/internal/ipfs"

	"github.com/decentraland/content-service/internal/auth"
	"github.com/decentraland/content-service/internal/metrics"
	"github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-cid"
	"github.com/ipsn/go-ipfs/gxlibs/github.com/ipfs/go-verifcid"

	"github.com/decentraland/content-service/internal/storage"
	"github.com/decentraland/content-service/internal/utils/rpc"
	log "github.com/sirupsen/logrus"
)

type UploadRequest struct {
	Proof   *entities.DeployProof     `validate:"required"`
	Mapping []entities.ContentMapping `validate:"required"`
	Content *RequestContent           `validate:"required"`
	Parcels []string                  `validate:"required"`
	Deploy  *entities.Deploy          `validate:"required"`
	Origin  string
}

type RequestContent struct {
	ContentFiles map[string][]*multipart.FileHeader
	RawDeploy    []*multipart.FileHeader
	RawProof     []*multipart.FileHeader
	RawMapping   []*multipart.FileHeader
}

func NewRequestContent(c map[string][]*multipart.FileHeader) (*RequestContent, error) {
	ret := &RequestContent{}
	files := make(map[string][]*multipart.FileHeader)
	for k, v := range c {
		if k != "deploy.json" && k != "proof.json" && k != "mapping.json" {
			files[k] = v
		}
	}
	ret.ContentFiles = files
	p, ok := c["proof.json"]
	if !ok {
		return nil, RequiredValueError{"missing proof.json"}
	}
	ret.RawProof = p

	d, ok := c["deploy.json"]
	if !ok {
		return nil, RequiredValueError{"missing deploy.json"}
	}
	ret.RawDeploy = d

	m, ok := c["mapping.json"]
	if !ok {
		return nil, RequiredValueError{"missing mapping.json"}
	}
	ret.RawMapping = m

	return ret, nil
}

// Groups all the files in the list by file CID
// The map will contain an entry for each CID, and the associated value would be a list of all the paths
func (r *UploadRequest) GroupFilePathsByCid() map[string][]string {
	filesPaths := make(map[string][]string)
	for _, fileMeta := range r.Mapping {
		paths := filesPaths[fileMeta.Cid]
		if paths == nil {
			paths = []string{}
		}
		filesPaths[fileMeta.Cid] = append(paths, fileMeta.Name)
	}
	return filesPaths
}

func (r *UploadRequest) CheckRequiredFiles() bool {
	for _, req := range r.Deploy.Required {
		_, ok := r.Content.ContentFiles[req.Cid]
		if !ok {
			return false
		}
	}
	return true
}

type UploadService interface {
	ProcessUpload(r *UploadRequest) error
}

type uploadService struct {
	Storage         storage.Storage
	IpfsHelper      *ipfs.IpfsHelper
	Auth            auth.Authorization
	Agent           *metrics.Agent
	ParcelSizeLimit int64
	rpc             rpc.RPC
	Log             *log.Logger
	MRepo           deployment.Repository
}

func NewUploadService(storage storage.Storage, helper *ipfs.IpfsHelper, auth auth.Authorization,
	agent *metrics.Agent, parcelSizeLimit int64, repo deployment.Repository,
	rpc rpc.RPC, l *log.Logger) UploadService {
	return &uploadService{
		Storage:         storage,
		IpfsHelper:      helper,
		Auth:            auth,
		Agent:           agent,
		ParcelSizeLimit: parcelSizeLimit,
		rpc:             rpc,
		Log:             l,
		MRepo:           repo,
	}
}

func (us *uploadService) ProcessUpload(r *UploadRequest) error {
	us.Log.Debug("Processing Upload request")

	if err := us.validateDeployment(r); err != nil {
		return err
	}

	if !r.CheckRequiredFiles() {
		return InvalidArgument{"missing required files"}
	}

	if err := us.validateSignature(us.Auth, r.Proof); err != nil {
		return err
	}

	if err := validateKeyAccess(us.Auth, r.Proof.Address, r.Parcels, us.Log); err != nil {
		return err
	}

	if err := us.validateRequestSize(r); err != nil {
		return err
	}

	t := time.Now()
	err := us.validateRequestContent(r.Content.ContentFiles, r.Mapping, r.Proof.ID)
	us.Agent.RecordUploadRequestValidationTime(time.Since(t))

	if err != nil {
		return err
	}

	pathsByCid := r.GroupFilePathsByCid()
	if err := us.processUploadedFiles(r.Content.ContentFiles, r.Proof.ID); err != nil {
		return err
	}

	if err := us.saveMapping(r); err != nil {
		return err
	}

	if err := us.MRepo.StoreDeployment(r.Deploy, r.Proof); err != nil {
		return err
	}

	if err := us.MRepo.StoreMapping(r.Proof.ID, r.Deploy.Positions); err != nil {
		return err
	}

	us.Agent.RecordUpload(r.Proof.ID, r.Proof.Address, r.Parcels, pathsByCid, r.Origin)

	return nil
}

func (us *uploadService) validateDeployment(r *UploadRequest) error {

	mCid, err := us.calculateCID(r.Content.RawMapping[0])
	if err != nil {
		return InvalidArgument{fmt.Sprintf("fail to calculate mapping.json CID: %s", err.Error())}
	}

	if r.Deploy.Mappings != mCid {
		return InvalidArgument{
			fmt.Sprintf("calculated mapping.json CID[%s] does not match:  %s", mCid, r.Deploy.Mappings)}
	}

	dCid, err := us.calculateCID(r.Content.RawDeploy[0])
	if err != nil {
		return InvalidArgument{fmt.Sprintf("fail to calculate deploy.json CID: %s", err.Error())}
	}

	if r.Proof.ID != dCid {
		return InvalidArgument{
			fmt.Sprintf("calculated deploy.json CID[%s] does not match:  %s", dCid, r.Proof.ID)}
	}

	return nil
}

// Retrieves an error if the signature is invalid, of if the signature does not corresponds to the given key and message
func (us *uploadService) validateSignature(a auth.Authorization, p *entities.DeployProof) error {
	us.Log.Debugf("Validating signature: %s", p.Signature)

	// ERC 1654 support https://github.com/ethereum/EIPs/issues/1654
	// We need to validate against a contract address whether this is ok or not?
	if len(p.Signature) > 150 {
		signature := p.Signature
		address := p.Address
		msg := fmt.Sprintf("%s.%d", p.ID, p.Timestamp)
		valid, err := us.rpc.ValidateDapperSignature(address, msg, signature)
		if err != nil {
			return err
		}
		if !valid {
			return fmt.Errorf("signature fails to verify for %s", address)
		}
		return nil
	}
	if !a.IsSignatureValid(fmt.Sprintf("%s.%d", p.ID, p.Timestamp), p.Signature, p.Address) {
		us.Log.Debugf("Invalid signature[%s] for SceneCid[%s] and pubKey[%s]", p.ID, p.Signature, p.Address)
		return InvalidArgument{"Signature is invalid"}
	}
	return nil
}

// Retrieves an error if the calculated global CID differs from the expected CID
func (us *uploadService) validateRequestContent(requestFiles map[string][]*multipart.FileHeader,
	mappings []entities.ContentMapping, sceneCID string) error {

	us.Log.Debugf("Validating content. SceneCID: %s", sceneCID)
	if err := checkCIDFormat(sceneCID, us.Log); err != nil {
		return err
	}
	for _, m := range mappings {
		if strings.HasSuffix(m.Name, "/") {
			continue
		}
		rFile, ok := requestFiles[m.Cid]
		if ok {
			fileCID, err := us.calculateCID(rFile[0])
			if err != nil {
				us.Log.Debugf("Failed to validate File[%s] cid: %s", m.Name, err.Error())
				return err
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

func (us *uploadService) calculateCID(file *multipart.FileHeader) (string, error) {
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
func validateKeyAccess(a auth.Authorization, pKey string, parcels []string, log *log.Logger) error {
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

func (us *uploadService) processUploadedFiles(fh map[string][]*multipart.FileHeader, cid string) error {
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

func (us *uploadService) validateRequestSize(r *UploadRequest) error {
	maxSize := int64(len(r.Parcels)) * us.ParcelSizeLimit

	size, err := us.estimateRequestSize(r)
	if err != nil {
		return err
	}

	if size > maxSize {
		us.Log.Errorf("UploadRequest ID[%s] exceeds the allowed limit Max[bytes]: %d, RequestSize[bytes]: %d",
			r.Proof.ID,
			maxSize,
			size)
		return InvalidArgument{
			fmt.Sprintf("UploadRequest exceeds the allowed limit Max[bytes]: %d, RequestSize[bytes]: %d",
				maxSize, size)}
	}
	return nil
}

func (us *uploadService) estimateRequestSize(r *UploadRequest) (int64, error) {
	size := int64(0)
	for _, m := range r.Mapping {
		if strings.HasSuffix(m.Name, "/") {
			continue
		}
		if f, ok := r.Content.ContentFiles[m.Cid]; ok {
			size += f[0].Size
		} else {
			s, err := us.Storage.FileSize(m.Cid)
			if err != nil {
				return 0, handleStorageError(err, m.Cid, us.Log)
			}
			size += s
		}
	}
	us.Log.Debugf(fmt.Sprintf("UploadRequest size: %d", size))
	return size, nil
}

func (us *uploadService) saveMapping(r *UploadRequest) error {
	file, err := r.Content.RawMapping[0].Open()
	if err != nil {
		us.Log.Errorf("Failed to store mapping[%s] ", r.Deploy.Mappings)
		return UnexpectedError{"fail to store file", err}
	}
	defer file.Close()

	if _, err := us.Storage.SaveFile(r.Deploy.Mappings, file, "application/json"); err != nil {
		return handleStorageError(err, r.Deploy.Mappings, us.Log)
	}
	return nil
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
