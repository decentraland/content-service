package main

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestUploadHandlerLocal(t *testing.T) {
	localStorage = true
	s3Storage = false

	multiBody, contentType := createMultipartBody()
	req := httptest.NewRequest("POST", "localhost:8000/mappings", multiBody)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	uploadHandler(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	var responseBody []uploadFile
	err := json.Unmarshal(body, &responseBody)
	if err != nil {
		t.Error("Error during unmarshal of response")
	}

	if resp.StatusCode != 200 {
		t.Errorf("Status code should be 200, but instead is %v", resp.StatusCode)
	}

	return
}

func TestUploadHandlerS3(t *testing.T) {
	localStorage = false
	s3Storage = true

	multiBody, contentType := createMultipartBody()
	req := httptest.NewRequest("POST", "localhost:8000/mappings", multiBody)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	uploadHandler(w, req)

	resp := w.Result()
	body, _ := ioutil.ReadAll(resp.Body)

	var responseBody []uploadFile
	err := json.Unmarshal(body, &responseBody)
	if err != nil {
		t.Error("Error during unmarshal of response")
	}

	if resp.StatusCode != 200 {
		t.Errorf("Status code should be 200, but instead is %v", resp.StatusCode)
	}

	return
}

// helper functions
func createMultipartBody() (io.Reader, string) {
	path := "main_test.go"
	file, err := os.Open(path)
	if err != nil {
		return nil, ""
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return nil, ""
	}
	_, err = io.Copy(part, file)

	err = writer.Close()
	if err != nil {
		return nil, ""
	}

	return body, writer.FormDataContentType()
}
