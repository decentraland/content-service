package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"flag"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gorilla/mux"
)

type uploadFile struct {
	Name string `json:"name"`
	Cid  string `json:"cid"`
}

var localStorage, s3Storage bool

func saveFile(fileDescriptor multipart.File, filename string) (string, error) {
	dst, err := os.Create("/tmp/" + filename)
	if err != nil {
		return "", err
	}

	_, err = io.Copy(dst, fileDescriptor)
	if err != nil {
		return "", err
	}

	return filename, nil
}

func getFileS3(cid string) string {
	return "https://content-service.s3.amazonaws.com/" + cid
}

func saveFileS3(fileDescriptor multipart.File, filename string) (string, error) {
	sess := session.Must(session.NewSession())

	uploader := s3manager.NewUploader(sess)

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String("content-service"),
		Key:    aws.String(filename),
		Body:   fileDescriptor,
	})
	if err != nil {
		fmt.Printf("failed to upload file, %v", err)
		return "", err
	}

	return result.Location, nil
}

func mappingsHandler(w http.ResponseWriter, r *http.Request) {
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseMultipartForm(0)
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

		var hash string
		if s3Storage {
			hash, err = saveFileS3(file, fileHeader.Filename)
		} else {
			hash, err = saveFile(file, fileHeader.Filename)
		}
		if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		savedFiles = append(savedFiles, uploadFile{fileHeader.Filename, hash})
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
	location := getFileS3(params["cid"])

	http.Redirect(w, r, location, 301)
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
}

func main() {
	flag.BoolVar(& localStorage, "local", true, "Local storage")
	flag.BoolVar(& s3Storage, "s3", false, "S3 storage")
	flag.Parse()

	if !localStorage && !s3Storage {
		fmt.Println("Need to set either local or s3 storage")
		return
	}

	r := mux.NewRouter()

	r.HandleFunc("/mappings", mappingsHandler).Methods("GET").Queries("x1", "{x1}", "y1", "{y1}", "x2", "{x2}", "y2", "{y2}")
	r.HandleFunc("/mappings", uploadHandler).Methods("POST")
	r.HandleFunc("/contents/{cid}", contentsHandler).Methods("GET")
	r.HandleFunc("/validate", validateHandler).Methods("GET").Queries("x", "{x}", "y", "{y}")

	log.Fatal(http.ListenAndServe(":8000", r))
}
