package handlers

import (
	"encoding/json"
	"errors"
	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/validation"
	"github.com/fatih/structs"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/decentraland/content-service/storage"
	"github.com/ipsn/go-ipfs/core"
	"github.com/ipsn/go-ipfs/core/coreunix"
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

type UploadRequest struct {
	Metadata      Metadata                           `validate:"required"`
	Manifest      []FileMetadata                     `validate:"required"`
	UploadedFiles map[string][]*multipart.FileHeader `validate:"required"`
	Scene         *scene                             `validate:"required"`
}

type UploadService interface {
	ProcessUpload(r *UploadRequest) error
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

	uploadedFiles := r.MultipartForm.File

	scene, err := getScene(uploadedFiles, v)
	if err != nil {
		return nil, err
	}

	request := UploadRequest{Metadata: metadata, Manifest: manifestContent, UploadedFiles: uploadedFiles, Scene: scene}
	err = v.ValidateStruct(request)
	if err != nil {
		return nil, WrapInBadRequestError(err)
	}
	return &request, nil
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
func getScene(files map[string][]*multipart.FileHeader, v validation.Validator) (*scene, error) {
	for _, header := range files {
		if header[0].Filename == "scene.json" {
			sceneFile, err := header[0].Open()
			if err != nil {
				return nil, WrapInBadRequestError(err)
			}
			return parseSceneJsonFile(sceneFile, v)
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

// Check if the expectedCID matches the actual CID for a given file
func fileMatchesCID(node *core.IpfsNode, fileHeader *multipart.FileHeader, expectedCID string) (bool, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return false, err
	}
	defer file.Close()

	actualCID, err := coreunix.Add(node, file)
	if err != nil {
		return false, err
	}

	return expectedCID == actualCID, nil
}

// Extracts a the list of FileMetadata from the Request
func getManifestContent(r *http.Request, v validation.Validator, cid string) ([]FileMetadata, error) {
	filesJSON, isset := r.MultipartForm.Value[cid]
	if !isset {
		return nil, NewBadRequestError("Missing contents part in multipart ")
	}
	return parseFilesMetadata(filesJSON[0], v)
}

// Parse a Json String into an array of FileMetadata
// Retrieves an error if the Json String is malformed or if a required field is missing
func parseFilesMetadata(metadataStr string, v validation.Validator) ([]FileMetadata, error) {
	var filesMeta []FileMetadata
	err := json.Unmarshal([]byte(metadataStr), &filesMeta)
	if err != nil {
		return nil, WrapInInternalError(err)
	}
	for _, element := range filesMeta {
		err = v.ValidateStruct(element)
		if err != nil {
			return nil, WrapInBadRequestError(err)
		}
	}
	return filesMeta, nil
}

// Gruops all the files in the list by file CID
// The map will cointain an entry for each CID, and the associated value would be a list of all the paths
func groupFilePathsByCid(files []FileMetadata) map[string][]string {
	filesPaths := make(map[string][]string)
	for _, fileMeta := range files {
		paths := filesPaths[fileMeta.Cid]
		if paths == nil {
			paths = []string{}
		}
		filesPaths[fileMeta.Cid] = append(paths, fileMeta.Name)
	}
	return filesPaths
}

// Upload Logic from this point on

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

func (s *UploadServiceImpl) ProcessUpload(r *UploadRequest) error {

	err := validateSignature(s.Auth, r.Metadata)
	if err != nil {
		return err
	}

	err = validateRootCid(s.IpfsNode, r.Metadata.RootCid, r.Manifest, r.UploadedFiles)
	if err != nil {
		return err
	}

	err = validateKeyAccess(s.Auth, r.Metadata.PubKey, r.Scene.Scene.Parcels)
	if err != nil {
		return err
	}

	err = processUploadedFiles(r.UploadedFiles, s.IpfsNode, groupFilePathsByCid(r.Manifest), s.RedisClient, r.Metadata.RootCid, s.Storage)
	if err != nil {
		return err
	}

	err = storeParcelsInformation(r.Metadata.RootCid, r.Scene.Scene.Parcels, s.RedisClient)
	if err != nil {
		return err
	}

	err = s.RedisClient.StoreMetadata(r.Metadata.RootCid, structs.Map(r.Metadata))
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
func validateRootCid(node *core.IpfsNode, expectedCID string, filesMeta []FileMetadata, files map[string][]*multipart.FileHeader) error {
	actualRootCID, err := calculateRootCid(node, expectedCID, filesMeta, files)
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
func calculateRootCid(node *core.IpfsNode, rootPath string, filesMeta []FileMetadata, files map[string][]*multipart.FileHeader) (string, error) {
	rootDir := filepath.Join("/tmp", rootPath)

	for _, meta := range filesMeta {
		if meta.Name[len(meta.Name)-1:] == "/" {
			continue
		}

		// This anonymous function would allow the defers to work properly
		// preventing resources from being piled up
		err := func() error {
			fileHeader := files[meta.Cid][0]
			dir := filepath.Join(rootDir, filepath.Dir(meta.Name))
			filePath := filepath.Join(dir, filepath.Base(meta.Name))

			err := os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return err
			}

			dst, err := os.Create(filePath)
			if err != nil {
				return err
			}
			defer dst.Close()

			file, err := fileHeader.Open()
			if err != nil {
				return err
			}
			defer file.Close()

			_, err = io.Copy(dst, file)
			if err != nil {
				return err
			}
			return nil
		}()
		if err != nil {
			return "", err
		}
	}

	return coreunix.AddR(node, rootDir)
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
func processUploadedFiles(fh map[string][]*multipart.FileHeader, n *core.IpfsNode, paths map[string][]string, rc data.RedisClient, cid string, s storage.Storage) error {
	for fileCID, fileHeaders := range fh {
		fileHeader := fileHeaders[0]

		fileMatches, err := fileMatchesCID(n, fileHeader, fileCID)
		if err != nil {
			return WrapInBadRequestError(err)
		} else if !fileMatches {
			return NewBadRequestError("File CID does not match its generated CID")
		}

		// This anonymous function would allow the defers to work properly
		// preventing resources from being piled up
		err = func() error {
			file, err := fileHeader.Open()
			if err != nil {
				return WrapInInternalError(err)
			}
			defer file.Close()

			_, err = s.SaveFile(fileCID, file)
			if err != nil {
				return WrapInInternalError(err)
			}
			return nil
		}()
		if err != nil {
			return err
		}

		for _, path := range paths[fileCID] {
			err = rc.StoreContent(cid, path, fileCID)
			if err != nil {
				return WrapInInternalError(err)
			}
		}

		if err = rc.AddCID(fileCID); err != nil {
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
