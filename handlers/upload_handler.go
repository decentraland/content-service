package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/decentraland/content-service/metrics"
	"github.com/decentraland/content-service/validation"
	log "github.com/sirupsen/logrus"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type UploadCtx struct {
	StructValidator validation.Validator
	Service         UploadService
	Agent           metrics.Agent
	Filter          *ContentTypeFilter
}

type FileMetadata struct {
	Cid  string `json:"cid" validate:"required"`
	Name string `json:"name" validate:"required"`
}

type Metadata struct {
	Value        string `json:"value" structs:"value " validate:"required"`
	Signature    string `json:"signature" structs:"signature" validate:"required,prefix=0x"`
	Validity     string `json:"validity" structs:"validity" validate:"required"`
	ValidityType int    `json:"validityType" structs:"validityType" validate:"gte=0"`
	Sequence     int    `json:"sequence" structs:"sequence" validate:"gte=0"`
	PubKey       string `json:"pubkey" structs:"pubkey" validate:"required,eth_addr"`
	RootCid      string `json:"root_cid" structs:"root_cid" validate:"required"`
}

type scene struct {
	Display        display     `json:"display"`
	Owner          string      `json:"owner"`
	Scene          sceneData   `json:"scene"`
	Communications commsConfig `json:"communications"`
	Main           string      `json:"main" validate:"required"`
}

type display struct {
	Title string `json:"title"`
}

type sceneData struct {
	EstateID int      `json:"estateId"`
	Parcels  []string `json:"parcels" validate:"required"`
	Base     string   `json:"base" validate:"required"`
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

func UploadContent(ctx interface{}, r *http.Request) (Response, error) {
	c, ok := ctx.(UploadCtx)
	if !ok {
		log.Fatal("Invalid Handler configuration")
		return nil, NewInternalError("Invalid Configuration")
	}
	sendRequestData(c.Agent, r)

	log.Debug("About to parse Upload request...")
	tParse := time.Now()
	uploadRequest, err := parseRequest(r, c.StructValidator, c.Agent, c.Filter)
	c.Agent.RecordUploadRequestParseTime(time.Since(tParse))
	log.Debug("Upload request parsed")

	if err != nil {
		return nil, err
	}

	tProcess := time.Now()
	err = c.Service.ProcessUpload(uploadRequest)
	c.Agent.RecordUploadProcessTime(time.Since(tProcess))

	if err != nil {
		return nil, err
	}

	return NewOkEmptyResponse(), nil
}

// Extracts all the information from the http request
// If any part is missing or is invalid it will retrieve an error
func parseRequest(r *http.Request, v validation.Validator, agent metrics.Agent, filter *ContentTypeFilter) (*UploadRequest, error) {
	err := r.ParseMultipartForm(0)
	if err != nil {
		log.Errorf("Invalid UploadContent request: %s", err.Error())
		return nil, NewInternalError(err.Error())
	}

	metadata, err := getMetadata(r, v)
	if err != nil {
		return nil, err
	}

	manifestContent, err := getManifestContent(r, v, metadata.RootCid)
	if err != nil {
		return nil, err
	}
	agent.RecordManifestSize(len(*manifestContent))

	uploadedFiles := r.MultipartForm.File
	agent.RecordUploadRequestFiles(len(uploadedFiles))
	if err := validateContentTypes(uploadedFiles, filter); err != nil {
		return nil, err
	}

	manifestSize := len(*manifestContent)
	requestFilesNumber := len(uploadedFiles)

	if requestFilesNumber > manifestSize {
		log.Debugf("Request contains too many files. Max expected: %d, found: %d", manifestSize, requestFilesNumber)
		return nil, NewBadRequestError("Request contains too many files")
	}

	scene, err := getScene(uploadedFiles, v)
	if err != nil {
		return nil, err
	}

	request := UploadRequest{Metadata: metadata, Manifest: manifestContent, UploadedFiles: uploadedFiles, Scene: scene}
	err = v.ValidateStruct(request)
	if err != nil {
		log.Debugf("Invalid UploadRequest: %s", err.Error())
		return nil, WrapInBadRequestError(err)
	}
	return &request, nil
}

func validateContentTypes(files map[string][]*multipart.FileHeader, filter *ContentTypeFilter) error {
	for _, v := range files {
		for _, f := range v {
			t := f.Header.Get("Content-Type")
			if !filter.IsAllowed(t) {
				return NewBadRequestError(fmt.Sprintf("Invalid  Content-type: %s File: %s", t, f.Filename))
			}
		}
	}
	return nil
}

// Extracts the request Metadata
func getMetadata(r *http.Request, v validation.Validator) (Metadata, error) {
	metaMultipart, isset := r.MultipartForm.Value["metadata"]
	if !isset {
		log.Error("Metadata not  found in UploadRequest")
		return Metadata{}, NewBadRequestError("Missing metadata part in multipart")
	}
	return parseSceneMetadata(metaMultipart[0], v)
}

// Parse a Json String into a Metadata
// Retrieves an error if the Json String is malformed or if a required field is missing
func parseSceneMetadata(mStr string, v validation.Validator) (Metadata, error) {
	var meta Metadata
	err := json.Unmarshal([]byte(mStr), &meta)
	if err != nil {
		log.Debugf("Invalid metadata content: %s", err.Error())
		return Metadata{}, WrapInBadRequestError(err)
	}
	meta.RootCid = strings.TrimPrefix(meta.Value, "/ipfs/")
	err = v.ValidateStruct(meta)
	if err != nil {
		log.Debugf("Invalid metadata content: %s", err.Error())
		return Metadata{}, WrapInBadRequestError(err)
	}
	return meta, nil
}

// Extract the scene information from the upload request
func getScene(files map[string][]*multipart.FileHeader, v validation.Validator) (*scene, error) {
	for _, header := range files {
		if header[0].Filename == "scene.json" {
			sceneFile, err := header[0].Open()
			if err != nil {
				log.Debugf("Invalid scene.json: %s", err.Error())
				return nil, WrapInBadRequestError(err)
			}
			return parseSceneJsonFile(sceneFile, v)
		}
	}
	log.Error("Missing scene.json")
	return nil, NewBadRequestError("Missing scene.json")
}

// Transform a io.Reader into a scene object
// Retrieves an error if the scene object is missing a required field is missing
func parseSceneJsonFile(file io.Reader, v validation.Validator) (*scene, error) {
	var sce scene
	err := json.NewDecoder(file).Decode(&sce)
	if err != nil {
		log.Debugf("Invalid scene.json content: %s", err.Error())
		return nil, WrapInBadRequestError(err)
	}
	err = v.ValidateStruct(sce)
	if err != nil {
		log.Debugf("Invalid scene.json content: %s", err.Error())
		return nil, WrapInBadRequestError(err)
	}
	return &sce, nil
}

// Extracts a the list of FileMetadata from the Request
func getManifestContent(r *http.Request, v validation.Validator, cid string) (*[]FileMetadata, error) {
	filesJSON, isset := r.MultipartForm.Value[cid]
	if !isset {
		log.Debug("Missing content in multipart")
		return nil, NewBadRequestError("Missing content in multipart ")
	}
	return parseFilesMetadata(filesJSON[0], v)
}

// Parse a Json String into an array of FileMetadata
// Retrieves an error if the Json String is malformed or if a required field is missing
func parseFilesMetadata(metadataStr string, v validation.Validator) (*[]FileMetadata, error) {
	var filesMeta *[]FileMetadata
	err := json.Unmarshal([]byte(metadataStr), &filesMeta)
	if err != nil {
		return nil, WrapInInternalError(err)
	}
	for _, element := range *filesMeta {
		err = v.ValidateStruct(element)
		if err != nil {
			return nil, WrapInBadRequestError(err)
		}
	}
	return filesMeta, nil
}

func sendRequestData(a metrics.Agent, r *http.Request) {
	s := r.Header.Get("Content-length")
	val, err := strconv.Atoi(s)
	if err != nil {
		log.Errorf("Failed to retrieve Content-length header: %s", err.Error())
	} else {
		a.RecordUploadReqSize(val)
	}
}
