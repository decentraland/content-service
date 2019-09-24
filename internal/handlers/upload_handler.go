package handlers

import (
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/decentraland/content-service/config"
	"github.com/decentraland/content-service/metrics"
	"github.com/decentraland/content-service/validation"
	log "github.com/sirupsen/logrus"
)

type UploadHandler interface {
	UploadContent(c *gin.Context)
}

func NewUploadHandler(v validation.Validator, us UploadService, a *metrics.Agent, f *ContentTypeFilter,
	limits config.Limits, ttl int64, l *log.Logger) UploadHandler {
	return &uploadHandlerImpl{
		StructValidator: v,
		Service:         us,
		Agent:           a,
		Filter:          f,
		Limits:          limits,
		TimeToLive:      ttl,
		Log:             l,
	}
}

type uploadHandlerImpl struct {
	StructValidator validation.Validator
	Service         UploadService
	Agent           *metrics.Agent
	Filter          *ContentTypeFilter
	Limits          config.Limits
	TimeToLive      int64
	Log             *log.Logger
}

type ContentMapping struct {
	Cid  string `json:"cid" validate:"required"`
	Name string `json:"name" validate:"required"`
}

type Metadata struct {
	Signature string `json:"signature" structs:"signature" validate:"required,prefix=0x"`
	PubKey    string `json:"pubkey" structs:"pubkey" validate:"required,eth_addr"`
	SceneCid  string `json:"scene_cid" structs:"root_cid" validate:"required"`
	Timestamp int64  `json:"timestamp" structs:"timestamp" validate:"gte=0"`
}

type scene struct {
	Display        display     `json:"display"`
	Owner          string      `json:"owner"`
	Scene          sceneData   `json:"scene"`
	Communications commsConfig `json:"communications"`
	Main           string      `json:"main" validate:"required"`
	Mappings       []ContentMapping    `json:"mappings" validate:"required"`
}

type display struct {
	Title string `json:"title"`
}

type sceneData struct {
	EstateID int      `json:"estateId"`
	Parcels  []string `json:"parcels" validate:"required"`
	Base     string   `json:"base" validate:"required"`
}

func (s *sceneData) UniqueParcels() []string {
	parcels := map[string]bool{}
	for _, p := range s.Parcels {
		parcels[p] = true
	}
	unique := make([]string, len(parcels))
	i := 0
	for k := range parcels {
		unique[i] = k
		i++
	}
	return unique
}

type commsConfig struct {
	Type       string `json:"type"`
	Signalling string `json:"signalling"`
}

type ContentTypeFilter struct {
	filterPattern string
}

// Retrieves a new content filter. If the lis is empty all content types will be allowed
func NewContentTypeFilter(types []string) *ContentTypeFilter {
	if len(types) == 0 {
		return &ContentTypeFilter{filterPattern: ".*"}
	}
	pattern := "(" + strings.Join(types, "?)|(") + "?)"
	return &ContentTypeFilter{filterPattern: pattern}
}

func (f *ContentTypeFilter) IsAllowed(t string) bool {
	r := regexp.MustCompile(f.filterPattern)
	return r.MatchString(t)
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
		case InvalidArgument:
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

	metadata, err := getMetadata(r, c.StructValidator, c.Log)
	if err != nil {
		return nil, err
	}

	if hasRequestExpired(metadata, c.TimeToLive) {
		c.Log.Debug("expired request")
		return nil, InvalidArgument{Message: "expired request"}
	}

	uploadedFiles := r.MultipartForm.File
	c.Agent.RecordUploadRequestFiles(len(uploadedFiles))
	if err := validateContentTypes(uploadedFiles, c.Filter); err != nil {
		return nil, err
	}

	scene, err := getScene(uploadedFiles, c.StructValidator, c.Log)
	if err != nil {
		return nil, err
	}

	filesPerScene := c.Limits.ParcelAssetsLimit
	manifestSize := len(scene.Mappings)
	c.Agent.RecordManifestSize(len(scene.Mappings))

	requestFilesNumber := len(uploadedFiles)
	if requestFilesNumber > manifestSize {
		c.Log.Debugf("Request contains too many files. Max expected: %d, found: %d", manifestSize, requestFilesNumber)
		return nil, InvalidArgument{Message: "request contains too many files"}
	}

	sceneParcels := scene.Scene.UniqueParcels()
	nUniqueParcels := len(sceneParcels)

	if nUniqueParcels != len(scene.Scene.Parcels) {
		return nil, InvalidArgument{Message: "Parcel list contains duplicate elements"}
	}

	sceneMaxElements := nUniqueParcels * filesPerScene
	if manifestSize > sceneMaxElements {
		c.Log.Debugf("Max Elements per scene exceeded. Max Value: %d, Got: %d, Owner: %s", filesPerScene, manifestSize, metadata.PubKey)
		return nil, InvalidArgument{Message: fmt.Sprintf("Max Elements per scene exceeded. Max Value: %d, Got: %d", filesPerScene, manifestSize)}
	}

	request := UploadRequest{
		Metadata: metadata,
		Mappings: scene.Mappings,
		UploadedFiles: uploadedFiles,
		Parcels: sceneParcels,
		Origin: r.Header.Get("x-upload-origin"),
	}

	err = c.StructValidator.ValidateStruct(request)
	if err != nil {
		c.Log.WithError(err).Debug("invalid UploadRequest")
		return nil, RequiredValueError{Message: err.Error()}
	}
	return &request, nil
}

func validateContentTypes(files map[string][]*multipart.FileHeader, filter *ContentTypeFilter) error {
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

// Extracts the request Metadata
func getMetadata(r *http.Request, v validation.Validator, log *log.Logger) (*Metadata, error) {
	metaMultipart, isset := r.MultipartForm.Value["metadata"]
	if !isset {
		log.Error("Metadata not  found in UploadRequest")
		return nil, RequiredValueError{"missing metadata part in multipart"}
	}
	return parseSceneMetadata(metaMultipart[0], v, log)
}

// Parse a Json String into a Metadata
// Retrieves an error if the Json String is malformed or if a required field is missing
func parseSceneMetadata(mStr string, v validation.Validator, log *log.Logger) (*Metadata, error) {
	var meta Metadata
	err := json.Unmarshal([]byte(mStr), &meta)
	if err != nil {
		log.WithError(err).Debug("invalid metadata content")
		return nil, InvalidArgument{"invalid metadata content"}
	}
	err = v.ValidateStruct(meta)
	if err != nil {
		log.WithError(err).Debug("invalid metadata content")
		return nil, InvalidArgument{"invalid metadata content"}
	}
	return &meta, nil
}

// Extract the scene information from the upload request
func getScene(files map[string][]*multipart.FileHeader, v validation.Validator, log *log.Logger) (*scene, error) {
	for _, header := range files {
		if header[0].Filename == "scene.json" {
			sceneFile, err := header[0].Open()
			if err != nil {
				log.WithError(err).Debug("Invalid scene.json")
				return nil, InvalidArgument{"invalid scene.json"}
			}
			return parseSceneJsonFile(sceneFile, v, log)
		}
	}
	log.Error("Missing scene.json")
	return nil, RequiredValueError{"missing scene.json"}
}

// Transform a io.Reader into a scene object
// Retrieves an error if the scene object is missing a required field is missing
func parseSceneJsonFile(file io.Reader, v validation.Validator, log *log.Logger) (*scene, error) {
	var sce scene
	err := json.NewDecoder(file).Decode(&sce)
	if err != nil {
		log.WithError(err).Debug("invalid scene.json content")
		return nil, InvalidArgument{"invalid scene.json content"}
	}
	err = v.ValidateStruct(sce)
	if err != nil {
		log.WithError(err).Debug("invalid scene.json content")
		return nil, InvalidArgument{"invalid scene.json content"}
	}
	return &sce, nil
}

// Extracts a the list of ContentMapping from the Request
func getManifestContent(r *http.Request, v validation.Validator, cid string, log *log.Logger) ([]ContentMapping, error) {
	filesJSON, isset := r.MultipartForm.Value[cid]
	if !isset {
		log.Debug("Missing content in multipart")
		return nil, RequiredValueError{"missing content in multipart"}
	}
	return parseFilesMetadata(filesJSON[0], v)
}

// Parse a Json String into an array of ContentMapping
// Retrieves an error if the Json String is malformed or if a required field is missing
func parseFilesMetadata(metadataStr string, v validation.Validator) ([]ContentMapping, error) {
	var filesMeta []ContentMapping
	err := json.Unmarshal([]byte(metadataStr), &filesMeta)
	if err != nil {
		return nil, InvalidArgument{"invalid manifest"}
	}
	for _, element := range filesMeta {
		err = v.ValidateStruct(element)
		if err != nil {
			return nil, InvalidArgument{err.Error()}
		}
	}
	return filesMeta, nil
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

func hasRequestExpired(m *Metadata, ttl int64) bool {
	epochNow := time.Now().Unix()

	return epochNow-m.Timestamp > ttl
}
