package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/ipsn/go-ipfs/core"
)

var server *httptest.Server

func TestMain(m *testing.M) {
	// Start server
	config := GetConfig()

	redisClient, err := initRedisClient(config)
	if err != nil {
		log.Fatal(err)
	}

	var ipfsNode *core.IpfsNode
	ipfsNode, err = initIpfsNode()
	if err != nil {
		log.Fatal(err)
	}

	// Run tests with S3 Storage
	config.S3Storage = true
	router := GetRouter(config, redisClient, ipfsNode)
	server = httptest.NewServer(router)
	code := m.Run()
	server.Close()

	// Run tests with Local Storage
	// config.S3Storage = false
	// config.LocalStorage = "tmp/"
	// router = GetRouter(config, redisClient, ipfsNode)
	// server = httptest.NewServer(router)
	// defer server.Close()
	// code = m.Run()

	os.Exit(code)
}

func getNoRedirectClient() *http.Client {
	// Configure http.Client to avoid following redirects
	client := server.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return client
}

type Link struct {
	A    xml.Name `xml:"a"`
	Href string   `xml:"href,attr"`
}

func TestContentsHandler(t *testing.T) {
	const CID = "123456789"

	client := getNoRedirectClient()
	response, err := client.Get(server.URL + "/contents/" + CID)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusMovedPermanently {
		t.Error("Contents handler should respond with status code 301. Recieved code: ", response.StatusCode)
	}

	link := new(Link)
	err3 := xml.NewDecoder(response.Body).Decode(link)
	if err3 != nil {
		t.Error("Error parsing response body")
		return
	}

	expected := "https://content-service.s3.amazonaws.com/" + CID
	if link.Href != expected {
		t.Errorf("Should redirect to %s. Recieved link to : %s", expected, link.Href)
	}
}

func TestInvalidCoordinates(t *testing.T) {
	x1, y1, x2, y2 := -999, 999, -999, 999
	query := fmt.Sprintf("/mappings?nw=%d,%d&se=%d,%d", x1, y1, x2, y2)

	client := getNoRedirectClient()
	response, err := client.Get(server.URL + query)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Error("Mappings handler should respond with status code 200. Recieved code: ", response.StatusCode)
	}

	if contentType := response.Header.Get("Content-Type"); contentType != "application/json" {
		t.Error("Mappings handler should return JSON file. Got 'Content-Type' :", contentType)
	}

	body, err2 := ioutil.ReadAll(response.Body)
	if err2 != nil {
		t.Error(err2)
	}
	bodyString := string(body)
	if bodyString != "{}" {
		t.Errorf("Mappings handler should return empty JSON when requesting invalid coordinates.\nRecieved:\n%s", bodyString)
	}
}

func TestCoordinatesNotCached(t *testing.T) {
	x1, y1, x2, y2 := -10, 10, 10, -10
	query := fmt.Sprintf("/mappings?nw=%d,%d&se=%d,%d", x1, y1, x2, y2)

	client := getNoRedirectClient()
	response, err := client.Get(server.URL + query)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Error("Mappings handler should respond with status code 200. Recieved code: ", response.StatusCode)
	}

	if contentType := response.Header.Get("Content-Type"); contentType != "application/json" {
		t.Error("Mappings handler should return JSON file. Got 'Content-Type' :", contentType)
	}

	body, err2 := ioutil.ReadAll(response.Body)
	if err2 != nil {
		t.Error(err2)
	}
	bodyString := string(body)
	if bodyString != "{}" {
		t.Errorf("Mappings handler should return empty JSON when coordinates not in cache.\nRecieved:\n%s", bodyString)
	}
}

func validateCoordinates(x int, y int) (*http.Response, error) {
	query := fmt.Sprintf("/validate?x=%d&y=%d", x, y)

	client := getNoRedirectClient()
	return client.Get(server.URL + query)
}

func TestValidateCoordinatesNotInCache(t *testing.T) {
	x, y := -10, 10
	response, err := validateCoordinates(x, y)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusNotFound {
		t.Error("Validate handler should respond with status code 400 when coordinates not in cache. Recieved code: ", response.StatusCode)
	}
}

func TestUploadHandler(t *testing.T) {
	const metadataFile = "testdata/metadata.json"
	const contentsFile = "testdata/contents.json"
	const dataFolder = "demo"

	req, err := newfileUploadRequest(metadataFile, contentsFile, dataFolder)
	if err != nil {
		t.Fatal(err)
	}

	client := getNoRedirectClient()
	response, err2 := client.Do(req)
	if err2 != nil {
		t.Fatal(err2)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Error("Upload unsuccessful. Got response code: ", response.StatusCode)
	}
}

type contents struct {
	CID  string `json:"cid"`
	Name string `json:"name"`
}

func newfileUploadRequest(metadataFile string, contentsFile string, dataFolder string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	var contentsJSON []contents
	c, err := os.Open(contentsFile)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	err = json.NewDecoder(c).Decode(&contentsJSON)
	if err != nil {
		return nil, err
	}

	for _, content := range contentsJSON {
		if content.Name[len(content.Name)-1:] == "/" {
			continue
		}
		dataPath := filepath.Join(dataFolder, content.Name)
		
		part, err := writer.CreateFormFile(content.CID, dataPath)
		if err != nil {
			log.Errorf("Unable to open %s", dataPath)
			return nil, err
		}

		var f *os.File
		f, err = os.Open(dataPath)
		if err != nil {
			return nil, err
		}
		_, err = io.Copy(part, f)
		if err != nil {
			return nil, err
		}
	}

	var metadataBytes []byte
	metadataBytes, err = ioutil.ReadFile(metadataFile)
	if err != nil {
		return nil, err
	}
	var contentsBytes []byte
	contentsBytes, err = ioutil.ReadFile(contentsFile)
	if err != nil {
		return nil, err
	}

	_ = writer.WriteField("metadata", string(metadataBytes))
	_ = writer.WriteField("contents", string(contentsBytes))

	err = writer.Close()
	if err != nil {
		return nil, err
	}

	req, err4 := http.NewRequest("POST", server.URL+"/mappings", body)
	if err4 != nil {
		return nil, err4
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, err4
}
