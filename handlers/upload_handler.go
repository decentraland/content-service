package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"mime/multipart"
	"os"
	"path/filepath"
	"errors"
	"strings"

	"github.com/fatih/structs"
	"github.com/go-redis/redis"
	"github.com/ipsn/go-ipfs/core"
	"github.com/ipsn/go-ipfs/core/coreunix"
)

type UploadHandler struct {
	S3Storage bool
	LocalStorage string
	RedisClient *redis.Client
	IpfsNode *core.IpfsNode
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

	match, err := rootCIDMatches(meta.RootCid, r.MultipartForm.File)
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
		log.Println(err)
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

	for partName, fileHeaders := range r.MultipartForm.File {
		fileHeader := fileHeaders[0]

		filepath, fileCID := getPathAndCID(partName, fileHeader.Filename)

		fileMatches, err := fileMatchesCID(config.IpfsNode, fileHeader, fileCID)
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

		if handler.S3Storage {
			_, err = saveFileS3(file, fileCID)
		} else {
			_, err = saveFile(file, handler.LocalStorage, fileCID)
		}
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		err = handler.RedisClient.HSet("content_"+meta.RootCid, filepath, fileCID).Err()
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

type metadata struct {
	Value        string `json:"value" structs:"value"`
	Signature    string `json:"signature" structs:"signature"`
	Validity     string `json:"validity" structs:"validity"`
	ValidityType string `json:"validityType" structs:"validityType"`
	Sequence     string `json:"sequence" structs:"sequence"`
	PubKey       string `json:"pubkey" structs:"pubkey"`
	RootCid      string `json:"-" structs:"rootcid"`
}

func getMetadata(jsonString []byte) (metadata, error) {
	var meta metadata
	err := json.Unmarshal(jsonString, &meta)
	if err != nil {
		return metadata{}, err
	}

	meta.RootCid = strings.TrimPrefix(meta.Value, "/ipfs/")
	return meta, nil
}

func rootCIDMatches(node *core.IpfsNode, rootCID string, files map[string][]*multipart.FileHeader) (bool, error) {
	actualRootCID, err := getRootCID(node, rootCID, files)
	if err != nil {
		return false, err
	}

	return rootCID == actualRootCID, nil
}

func getRootCID(node *core.IpfsNode, rootCID string, files map[string][]*multipart.FileHeader) (string, error) {
	rootDir := filepath.Join("/tmp", rootCID)

	for path, fileHeaders := range files {
		fileHeader := fileHeaders[0]
		dir := filepath.Join(rootDir, filepath.Dir(path))
		filePath := filepath.Join(dir, fileHeader.Filename)

		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return "", err
		}

		dst, err := os.Create(filePath)
		if err != nil {
			return "", err
		}
		defer dst.Close()

		file, err := fileHeader.Open()
		if err != nil {
			return "", err
		}
		defer file.Close()

		_, err = io.Copy(dst, file)
		if err != nil {
			return "", err
		}
	}

	return coreunix.AddR(node, rootDir)
}

type scene struct {
	Display struct {
		Title string `json:"title"`
	} `json:"display"`
	// Contact <T> `json:"contact"`
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
	// Policy <T> `json:"policy"`
	Main string `json:"main"`
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

func getPathAndCID(part, filename string) (string, string) {
	var filepath string

	path := strings.Split(part, "/")
	fileCID := path[len(path)-1]

	if len(path) > 1 {
		filepath = strings.Join(path[:len(path)-1], "/") + "/" + filename
	} else {
		filepath = filename
	}

	return filepath, fileCID
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
