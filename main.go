package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/gorilla/mux"
)

type uploadFile struct {
	Name string `json:"name"`
	Cid  string `json:"cid"`
}

func saveFile(fileHeader *multipart.FileHeader) (string, error) {
	hash := sha256.Sum256([]byte(fileHeader.Filename))
	hashstr := hex.EncodeToString(hash[:])

	dst, err := os.Create("/tmp/" + string(hashstr))
	if err != nil {
		return "", err
	}

	file, err := fileHeader.Open()
	if err != nil {
		return "", err
	}

	_, err = io.Copy(dst, file)
	if err != nil {
		return "", err
	}

	return hashstr, nil
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

		hash, err := saveFileS3(file, fileHeader.Filename)
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
	r := mux.NewRouter()

	r.HandleFunc("/mappings", mappingsHandler).Methods("GET").Queries("x1", "{x1}", "y1", "{y1}", "x2", "{x2}", "y2", "{y2}")
	r.HandleFunc("/mappings", uploadHandler).Methods("POST")
	r.HandleFunc("/contents/{cid}", contentsHandler).Methods("GET")
	r.HandleFunc("/validate", validateHandler).Methods("GET").Queries("x", "{x}", "y", "{y}")

	log.Fatal(http.ListenAndServe(":8000", r))
}
