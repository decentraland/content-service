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

	"github.com/decentraland/content-service/handlers"
	"github.com/ipsn/go-ipfs/core"
)

var server *httptest.Server

func TestMain(m *testing.M) {
	// Start server
	config := GetConfig("config_test")

	redisClient, err := initRedisClient(config)
	if err != nil {
		log.Fatal(err)
	}

	var ipfsNode *core.IpfsNode
	ipfsNode, err = initIpfsNode()
	if err != nil {
		log.Fatal(err)
	}

	router := GetRouter(config, redisClient, ipfsNode)
	server = httptest.NewServer(router)
	defer server.Close()
	code := m.Run()

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

func TestContentsHandlerS3Redirect(t *testing.T) {
	const CID = "123456789"

	client := getNoRedirectClient()
	response, err := client.Get(server.URL + "/contents/" + CID)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	awsKeys := [3]string{"AWS_REGION", "AWS_ACCESS_KEY", "AWS_SECRET_KEY"}
	for _, key := range awsKeys {
		_, ok := os.LookupEnv(key)
		if !ok {
			t.Skip("S3 Storage disabled. Skipping test")
		}
	}

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
	x1, y1, x2, y2 := 45, 45, 44, 46
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
	x1, y1, x2, y2 := 120, 120, 120, 120
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
	const metadataFile = "test/data/metadata.json"
	const contentsFile = "test/data/contents.json"
	const dataFolder = "test/data/demo"

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

	// Test downloading test.txt
	const testFileCID = "QmbdQuGbRFZdeqmK3PJyLV3m4p2KDELKRS4GfaXyehz672"
	resp, err2 := client.Get(server.URL + "/contents/" + testFileCID)
	if err != nil {
		t.Fatal(err2)
	}
	testContents, err3 := ioutil.ReadAll(resp.Body)
	if err3 != nil {
		t.Fatal(err3)
	}
	if string(testContents) != "something\n" {
		t.Errorf("Test file contents do not match.\nExpected 'something'\nGot %s", string(testContents))
	}

	// Test validate handler
	x, y := 54, -136
	response, err4 := validateCoordinates(x, y)
	if err != nil {
		t.Fatal(err4)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Error("Validate handler should respond with status code 200. Recieved code: ", response.StatusCode)
	}
}

func newfileUploadRequest(metadataFile string, contentsFile string, dataFolder string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	var contentsJSON []handlers.FileMetadata
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

		part, err := writer.CreateFormFile(content.Cid, filepath.Base(content.Name))
		if err != nil {
			return nil, err
		}

		dataPath := filepath.Join(dataFolder, content.Name)
		var f *os.File
		f, err = os.Open(dataPath)
		if err != nil {
			log.Printf("Cannot open %s", dataPath)
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
	rootCID := getRootCID(metadataFile)
	_ = writer.WriteField(rootCID, string(contentsBytes))

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

func getRootCID(metadataFile string) string {
	var meta handlers.Metadata
	m, err := os.Open(metadataFile)
	if err != nil {
		log.Fatal(err)
	}
	defer m.Close()
	err = json.NewDecoder(m).Decode(&meta)
	if err != nil {
		log.Fatal(err)
	}
	return meta.Value
}