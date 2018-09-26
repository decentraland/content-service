package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

type uploadFile struct {
	Name string `json:"name"`
	Cid  string `json:"cid"`
}

type signature struct {
	Value        string `json:"value"`
	Signature    string `json:"signature"`
	Validity     string `json:"validity"`
	ValidityType string `json:"validityType"`
	Sequence     string `json:"sequence"`
}

type metadata struct {
	RootCid   string `json:"rootcid"`
	Timestamp string `json:"timestamp"`
}

func mappingsHandler(w http.ResponseWriter, r *http.Request) {
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(0)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	var sig signature
	err = json.Unmarshal([]byte(r.MultipartForm.Value["signature"][0]), &sig)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	var meta metadata
	err = json.Unmarshal([]byte(r.MultipartForm.Value["metadata"][0]), &meta)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	var savedFiles []uploadFile

	for _, fileHeaders := range r.MultipartForm.File {
		fileHeader := fileHeaders[0]

		file, err := fileHeader.Open()
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		var name string
		if s3Storage {
			name, err = saveFileS3(file, fileHeader.Filename)
		} else {
			name, err = saveFile(file, fileHeader.Filename)
		}
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		savedFiles = append(savedFiles, uploadFile{fileHeader.Filename, name})
	}

	err = json.NewEncoder(w).Encode(savedFiles)
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
