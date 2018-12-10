package handlers

import (
	"bytes"
	"encoding/json"
	"github.com/decentraland/content-service/validation"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
)

type UploadCtx struct {
	StructValidator validation.Validator
	Service         UploadService
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

func UploadContent(ctx interface{}, r *http.Request) (Response, error) {
	c, ok := ctx.(UploadCtx)
	if !ok {
		return nil, NewInternalError("Invalid Configuration")
	}

	uploadRequest, err := parseRequest(r, c.StructValidator)
	if err != nil {
		return nil, err
	}

	err = c.Service.ProcessUpload(uploadRequest)

	if err != nil {
		return nil, err
	}

	return NewOkEmptyResponse(), nil
}

// Extracts all the information from the http request
// If any part is missing or is invalid it will retrieve an error
func parseRequest(r *http.Request, v validation.Validator) (*UploadRequest, error) {
	err := r.ParseMultipartForm(0)
	if err != nil {
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

	if len(r.MultipartForm.File) > len(*manifestContent) {
		return nil, NewBadRequestError("Request contains too many files")
	}

	requestFiles := make(map[string]*FileContent)
	for k, v := range r.MultipartForm.File {
		h := v[0]
		c, err := getContent(h)
		if err != nil {
			return nil, err
		}
		requestFiles[k] = c
	}

	scene, err := getScene(requestFiles, v)
	if err != nil {
		return nil, err
	}

	request := UploadRequest{Metadata: metadata, Manifest: manifestContent, UploadedFiles: requestFiles, Scene: scene}
	err = v.ValidateStruct(request)
	if err != nil {
		return nil, WrapInBadRequestError(err)
	}
	return &request, nil
}

func getContent(h *multipart.FileHeader) (*FileContent, error) {
	file, err := h.Open()
	if err != nil {
		return nil, err
	}
	defer file.Close()

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(file)
	if err != nil {
		return nil, err
	}

	return &FileContent{FileName: h.Filename, Content: buf.Bytes()}, nil
}

// Extracts the request Metadata
func getMetadata(r *http.Request, v validation.Validator) (Metadata, error) {
	metaMultipart, isset := r.MultipartForm.Value["metadata"]
	if !isset {
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
		return Metadata{}, WrapInBadRequestError(err)
	}
	meta.RootCid = strings.TrimPrefix(meta.Value, "/ipfs/")
	err = v.ValidateStruct(meta)
	if err != nil {
		return Metadata{}, WrapInBadRequestError(err)
	}
	return meta, nil
}

// Extract the scene information from the upload request
func getScene(files map[string]*FileContent, v validation.Validator) (*scene, error) {
	for _, header := range files {
		if header.FileName == "scene.json" {
			return parseSceneJsonFile(header.Reader(), v)
		}
	}
	return nil, NewBadRequestError("Missing scene.json")
}

// Transform a io.Reader into a scene object
// Retrieves an error if the scene object is missing a required field is missing
func parseSceneJsonFile(file io.Reader, v validation.Validator) (*scene, error) {
	var sce scene
	err := json.NewDecoder(file).Decode(&sce)
	if err != nil {
		return nil, WrapInBadRequestError(err)
	}
	err = v.ValidateStruct(sce)
	if err != nil {
		return nil, WrapInBadRequestError(err)
	}
	return &sce, nil
}

// Extracts a the list of FileMetadata from the Request
func getManifestContent(r *http.Request, v validation.Validator, cid string) (*[]FileMetadata, error) {
	filesJSON, isset := r.MultipartForm.Value[cid]
	if !isset {
		return nil, NewBadRequestError("Missing contents part in multipart ")
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
