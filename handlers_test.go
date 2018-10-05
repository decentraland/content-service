package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
)

var server *http.Server
var serverURL string

func TestMain(m *testing.M) {
	// Start server
	setConfigParams()
	serverURL = "localhost:8000"
	router := getRouter()
	server = &http.Server{Addr: ":8000", Handler: router}
	err := server.ListenAndServe()
	if err != nil {
		panic("Cannot start test server")
	}

	code := m.Run()
	os.Exit(code)
}

func ProcessRequest(t *testing.T, method string, route string, body io.Reader) (*http.Response, error) {
	request, err := http.NewRequest(method, serverURL+route, body)
	if err != nil {
		t.Fatal(err)
	}

	httpClient := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		}}
	return httpClient.Do(request)
}

func TestHandleFakeCID(t *testing.T) {
	t.Log("Test server url is", serverURL)
	t.Log("s3Storage is", s3Storage)
	response, err := ProcessRequest(t, "GET", "/content/000", nil)

	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusPermanentRedirect {
		t.Error("Fake CID should return status code 301. Got", response.StatusCode)
	}
}

func TestHandleValidCID(t *testing.T) {
	cid := "106f1557-4a92-41a4-9f18-40fcb90b4031" // TODO: Find a valid CID
	response, err := ProcessRequest(t, "GET", "/content/"+cid, nil)
	defer response.Body.Close()

	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusPermanentRedirect {
		t.Error("Valid CID should return status code 301. Got", response.StatusCode)
	}

}

func TestValidateHandlerReturnsJSON(t *testing.T) {
	query := "x=32&y=-22"
	response, err := ProcessRequest(t, "GET", "/validate?"+query, nil)
	defer response.Body.Close()

	if err != nil {
		t.Fatal(err)
	}

	if contentType := response.Header.Get("Content-Type"); contentType != "application/json" {
		t.Error("Validate Handler should return JSON file. Got 'Content-Type' :", contentType)
	}
}

func TestMappingsHandlerWithFakeParcel(t *testing.T) {
	x1, y1, x2, y2 := -999, 999, -999, 999
	query := fmt.Sprintf("mappings?nw=%d,%d&se=%d,%d", x1, y1, x2, y2)
	response, err := ProcessRequest(t, "GET", query, nil)

	if err != nil {
		t.Fatal(err)
	}

	if response.StatusCode != http.StatusInternalServerError {
		t.Error("Valid CID should return status code 500. Got", response.StatusCode)
	}
}
