package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/fatih/structs"
	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

type uploadFile struct {
	Name string `json:"name"`
	Cid  string `json:"cid"`
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

func mappingsHandler(w http.ResponseWriter, r *http.Request) {
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	parcelID := fmt.Sprintf("%+v,%+v", params["x"], params["y"])

	parcelMeta, err := getParcelMetadata(parcelID)
	if err == redis.Nil {
		http.Error(w, http.StatusText(404), 404)
		return
	} else if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	metadataJSON, err := json.Marshal(parcelMeta)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	_, err = w.Write(metadataJSON)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
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

		fileMatches, err := fileMatchesCID(fileHeader, fileCID)
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

		if s3Storage {
			_, err = saveFileS3(file, fileCID)
		} else {
			_, err = saveFile(file, fileCID)
		}
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		err = client.HSet("content_"+meta.RootCid, filepath, fileCID).Err()
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}
	}

	for _, parcel := range scene.Scene.Parcels {
		err = client.Set(parcel, meta.RootCid, 0).Err()
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}
	}

	err = client.HMSet("metadata_"+meta.RootCid, structs.Map(meta)).Err()
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
}

func contentsHandler(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	if s3Storage {
		location := getFileS3(params["cid"])
		http.Redirect(w, r, location, 301)
	} else {
		location := getFile(params["cid"])
		w.Header().Add("Content-Disposition", "Attachment")
		http.ServeFile(w, r, location)
	}
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

func fileMatchesCID(fileHeader *multipart.FileHeader, CID string) (bool, error) {
	file, err := fileHeader.Open()
	if err != nil {
		return false, err
	}
	defer file.Close()

	filecontent, err := ioutil.ReadAll(file)
	if err != nil {
		return false, err
	}

	fileCID, err := cidPref.Sum(filecontent)
	if err != nil {
		return false, err
	}

	return fileCID.String() == CID, nil
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
