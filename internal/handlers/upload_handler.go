package handlers

import (
	"encoding/json"
	"fmt"
	"mime/multipart"
	"net/http"
	"strconv"
	"time"

	"github.com/decentraland/content-service/internal/entities"

	"github.com/decentraland/content-service/utils"

	"github.com/gin-gonic/gin"

	"github.com/decentraland/content-service/metrics"
	"github.com/decentraland/content-service/validation"
	log "github.com/sirupsen/logrus"
)

type UploadHandler interface {
	UploadContent(c *gin.Context)
}

func NewUploadHandler(v validation.Validator, us UploadService, a *metrics.Agent, f utils.ContentTypeFilter,
	limit int, ttl int64, l *log.Logger) UploadHandler {
	return &uploadHandlerImpl{
		StructValidator:  v,
		Service:          us,
		Agent:            a,
		Filter:           f,
		ParcelAssetLimit: limit,
		TimeToLive:       ttl,
		Log:              l,
	}
}

type uploadHandlerImpl struct {
	StructValidator  validation.Validator
	Service          UploadService
	Agent            *metrics.Agent
	Filter           utils.ContentTypeFilter
	ParcelAssetLimit int
	TimeToLive       int64
	Log              *log.Logger
}

func (uh *uploadHandlerImpl) UploadContent(c *gin.Context) {
	sendRequestData(uh.Agent, c.Request, uh.Log)

	uh.Log.Debug("About to parse Upload request...")
	tParse := time.Now()
	uploadRequest, err := uh.parseRequest(c.Request)
	uh.Agent.RecordUploadRequestParseTime(time.Since(tParse))
	log.Debug("Upload request parsed")

	if err != nil {
		uh.Log.WithError(err).Error("Error parsing upload")
		switch e := err.(type) {
		case InvalidArgument, RequiredValueError:
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": e.Error()})
			return
		default:
			_ = c.Error(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error, try again later"})
			return
		}
	}

	tProcess := time.Now()
	err = uh.Service.ProcessUpload(uploadRequest)
	uh.Agent.RecordUploadProcessTime(time.Since(tProcess))

	if err != nil {
		uh.Log.WithError(err).Error("Error parsing upload")
		switch e := err.(type) {
		case InvalidArgument:
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": e.Error()})
			return
		case UnauthorizedError:
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": e.Error()})
			return
		default:
			_ = c.Error(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "internal error, try again later"})
			return
		}
	}

	c.Status(http.StatusOK)
}

// Extracts all the information from the http request
// If any part is missing or is invalid it will retrieve an error
func (c *uploadHandlerImpl) parseRequest(r *http.Request) (*UploadRequest, error) {
	err := r.ParseMultipartForm(0)
	if err != nil {
		c.Log.WithError(err).Error("Invalid UploadContent request")
		return nil, UnexpectedError{"error parsing request form", err}
	}

	content, err := NewRequestContent(r.MultipartForm.File)
	if err != nil {
		c.Log.WithError(err).Error("Invalid UploadContent request")
		switch err.(type) {
		case RequiredValueError:
			return nil, err
		default:
			return nil, UnexpectedError{"error parsing request form", err}
		}
	}

	proof, err := c.getDeploymentProof(content)
	if err != nil {
		return nil, err
	}

	if hasRequestExpired(proof.Timestamp, c.TimeToLive) {
		c.Log.Debug("expired request")
		return nil, InvalidArgument{Message: "expired request"}
	}

	deploy, err := c.getDeploy(content)
	if err != nil {
		return nil, err
	}

	if hasRequestExpired(deploy.Timestamp, c.TimeToLive) {
		c.Log.Debug("expired request")
		return nil, InvalidArgument{Message: "expired request"}
	}

	mapping, err := c.getMappings(content)
	if err != nil {
		return nil, err
	}

	filesPerScene := c.ParcelAssetLimit
	mSize := len(mapping)
	c.Agent.RecordManifestSize(mSize)

	requestFilesNumber := len(content.ContentFiles)
	if requestFilesNumber > mSize {
		c.Log.Debugf("Request contains too many files. Max expected: %d, found: %d", mSize, requestFilesNumber)
		return nil, InvalidArgument{Message: "request contains too many files"}
	}

	sceneParcels := deploy.UniquePositions()
	nUniqueParcels := len(sceneParcels)

	if nUniqueParcels != len(deploy.Positions) {
		return nil, InvalidArgument{Message: "Parcel list contains duplicate elements"}
	}

	sceneMaxElements := nUniqueParcels * filesPerScene
	if mSize > sceneMaxElements {
		c.Log.Debugf("Max Elements per scene exceeded. Max Value: %d, Got: %d, Owner: %s", filesPerScene, mSize, proof.Address)
		return nil, InvalidArgument{Message: fmt.Sprintf("Max Elements per scene exceeded. Max Value: %d, Got: %d", filesPerScene, mSize)}
	}

	c.Agent.RecordUploadRequestFiles(len(content.ContentFiles))
	if err := validateContentTypes(content.ContentFiles, c.Filter); err != nil {
		return nil, err
	}
	request := UploadRequest{
		Proof:   proof,
		Mapping: mapping,
		Content: content,
		Parcels: sceneParcels,
		Deploy:  deploy,
		Origin:  r.Header.Get("x-upload-origin"),
	}

	err = c.StructValidator.ValidateStruct(request)
	if err != nil {
		c.Log.WithError(err).Debug("invalid UploadRequest")
		return nil, RequiredValueError{Message: err.Error()}
	}
	return &request, nil
}

func validateContentTypes(files map[string][]*multipart.FileHeader, filter utils.ContentTypeFilter) error {
	for _, v := range files {
		for _, f := range v {
			t := f.Header.Get("Content-Type")
			if !filter.IsAllowed(t) {
				return InvalidArgument{Message: fmt.Sprintf("Invalid  Content-type: %s File: %s", t, f.Filename)}
			}
		}
	}
	return nil
}

func (uh *uploadHandlerImpl) getDeploymentProof(content *RequestContent) (*entities.DeployProof, error) {
	var proof entities.DeployProof
	err := uh.getEntityFromMultipart(content.RawProof, &proof, nil)
	if err != nil {
		return nil, err
	}
	return &proof, nil
}

func (uh *uploadHandlerImpl) getDeploy(content *RequestContent) (*entities.Deploy, error) {
	var d entities.Deploy
	err := uh.getEntityFromMultipart(content.RawDeploy, &d, nil)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (uh *uploadHandlerImpl) getMappings(content *RequestContent) ([]entities.ContentMapping, error) {
	var m []entities.ContentMapping
	err := uh.getEntityFromMultipart(content.RawMapping, &m, func(validator validation.Validator, e interface{}) error {
		val := e.(*[]entities.ContentMapping)
		for _, cm := range *val {
			errVal := validator.ValidateStruct(cm)
			if errVal != nil {
				return errVal
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return m, nil
}

// Extracts the request Metadata
func (c *uploadHandlerImpl) getEntityFromMultipart(fh []*multipart.FileHeader, entity interface{},
	validate func(validator validation.Validator, entity interface{}) error) error {
	file, err := fh[0].Open()
	if err != nil {
		log.WithError(err).Debug("Invalid content")
		return InvalidArgument{"invalid content"}
	}
	err = json.NewDecoder(file).Decode(entity)
	if err != nil {
		log.WithError(err).Debug("invalid content")
		return InvalidArgument{"invalid content"}
	}

	if validate == nil {
		err = c.StructValidator.ValidateStruct(entity)
	} else {
		err = validate(c.StructValidator, entity)
	}

	if err != nil {
		log.WithError(err).Debug("invalid content")
		return InvalidArgument{"invalid content"}
	}
	return nil
}

func sendRequestData(a *metrics.Agent, r *http.Request, log *log.Logger) {
	s := r.Header.Get("Content-length")
	val, err := strconv.Atoi(s)
	if err != nil {
		log.Errorf("Failed to retrieve Content-length header: %s", err.Error())
	} else {
		a.RecordUploadReqSize(val)
	}
}

func hasRequestExpired(t int64, ttl int64) bool {
	epochNow := time.Now().Unix()

	return epochNow-t > ttl
}
