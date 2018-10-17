package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"


	"github.com/decentraland/content-service/storage"
	"github.com/fatih/structs"
	"github.com/go-redis/redis"
	"github.com/ipsn/go-ipfs/core"
	"github.com/ipsn/go-ipfs/core/coreunix"
)

type UploadHandler struct {
	Storage storage.Storage
	RedisClient  *redis.Client
	IpfsNode     *core.IpfsNode
}

type FileMetadata struct {
	Cid  string `json:"cid"`
	Name string `json:"name"`
}

type Metadata struct {
	Value        string `json:"value" structs:"value"`
	Signature    string `json:"signature" structs:"signature"`
	Validity     string `json:"validity" structs:"validity"`
	ValidityType int `json:"validityType" structs:"validityType"`
	Sequence     int `json:"sequence" structs:"sequence"`
	PubKey       string `json:"pubkey" structs:"pubkey"`
	RootCid      string `json:"root_cid" structs:"root_cid"`
}

type scene struct {
	Display struct {
		Title string `json:"title"`
	} `json:"display"`
	Owner string `json:"owner"`
	Scene struct {
		EstateID int      `json:"estateId"`
		Parcels  []string `json:"parcels"`
		Base     string   `json:"base"`
	} `json:"scene"`
	Communications struct {
		Type       string `json:"type"`
		Signalling string `json:"signalling"`
	} `json:"communications"`
	Main string `json:"main"`
}

func (handler *UploadHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(0)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	metaMultipart, isset := r.MultipartForm.Value["metadata"]
	if !isset {
		log.Println(err)
		http.Error(w, http.StatusText(400), 400)
		return
	}

	meta, err := getMetadata([]byte(metaMultipart[0]))
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	valid, err := isSignatureValid(meta.RootCid, meta.Signature, meta.PubKey)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	} else if !valid {
		http.Error(w, http.StatusText(401), 401)
		return
	}

	filesJSON, isset := r.MultipartForm.Value[meta.RootCid]
	if !isset {
		http.Error(w, http.StatusText(400), 400)
		return
	}

	var filesMeta []FileMetadata
	err = json.Unmarshal([]byte(filesJSON[0]), &filesMeta)
	if err != nil {
		http.Error(w, http.StatusText(500), 500)
		return
	}

	filesPath := make(map[string]string)
	for _, fileMeta := range filesMeta {
		filesPath[fileMeta.Cid] = fileMeta.Name
	}

	match, err := rootCIDMatches(handler.IpfsNode, meta.RootCid, filesMeta, r.MultipartForm.File)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	} else if !match {
		http.Error(w, http.StatusText(400), 400)
		return
	}

	scene, err := getScene(r.MultipartForm.File)
	if err != nil {
		http.Error(w, http.StatusText(400), 400)
		return
	}

	canModify, err := userCanModify(meta.PubKey, scene)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	} else if !canModify {
		http.Error(w, http.StatusText(401), 401)
		return
	}

	for fileCID, fileHeaders := range r.MultipartForm.File {
		fileHeader := fileHeaders[0]

		fileMatches, err := fileMatchesCID(handler.IpfsNode, fileHeader, fileCID)
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(500), 500)
			return
		} else if !fileMatches {
			http.Error(w, http.StatusText(400), 400)
			return
		}

		file, err := fileHeader.Open()
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}
		defer file.Close()

		_, err = handler.Storage.SaveFile(fileCID, file)
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		err = handler.RedisClient.HSet("content_"+meta.RootCid, filesPath[fileCID], fileCID).Err()
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}
	}

	for _, parcel := range scene.Scene.Parcels {
		err = handler.RedisClient.Set(parcel, meta.RootCid, 0).Err()
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}
	}

	err = handler.RedisClient.HMSet("metadata_"+meta.RootCid, structs.Map(meta)).Err()
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
}

func getMetadata(jsonString []byte) (Metadata, error) {
	var meta Metadata
	err := json.Unmarshal(jsonString, &meta)
	if err != nil {
		return Metadata{}, err
	}

	meta.RootCid = strings.TrimPrefix(meta.Value, "/ipfs/")
	return meta, nil
}

func rootCIDMatches(node *core.IpfsNode, rootCID string, filesMeta []FileMetadata, files map[string][]*multipart.FileHeader) (bool, error) {
	rootDir := filepath.Join("/tmp", rootCID)

	for _, meta := range filesMeta {
		if meta.Name[len(meta.Name)-1:] == "/" {
			continue
		}

		fileHeader := files[meta.Cid][0]
		dir := filepath.Join(rootDir, filepath.Dir(meta.Name))
		filePath := filepath.Join(dir, fileHeader.Filename)

		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return false, err
		}

		dst, err := os.Create(filePath)
		if err != nil {
			return false, err
		}
		defer dst.Close()

		file, err := fileHeader.Open()
		if err != nil {
			return false, err
		}
		defer file.Close()

		_, err = io.Copy(dst, file)
		if err != nil {
			return false, err
		}
	}

	actualRootCID, err := coreunix.AddR(node, rootDir)
	if err != nil {
		return false, err
	}

	return rootCID == actualRootCID, nil
}

func getScene(files map[string][]*multipart.FileHeader) (*scene, error) {
	for _, header := range files {
		if header[0].Filename == "scene.json" {
			sceneFile, err := header[0].Open()
			if err != nil {
				return nil, err
			}

			var sce scene
			err = json.NewDecoder(sceneFile).Decode(&sce)
			if err != nil {
				return nil, err
			}

			return &sce, nil
		}
	}

	return nil, errors.New("Missing scene.json")
}

func fileMatchesCID(node *core.IpfsNode, fileHeader *multipart.FileHeader, receivedCID string) (bool, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return false, err
	}
	defer file.Close()

	actualCID, err := coreunix.Add(node, file)
	if err != nil {
		return false, err
	}

	return receivedCID == actualCID, nil
}
