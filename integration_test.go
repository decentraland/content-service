package main

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/stretchr/testify/assert"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/decentraland/content-service/config"
	"github.com/decentraland/content-service/handlers"
)

type uploadTestConfig struct {
	name           string
	metadataPath   string
	contentDir     string
	manifest       string
	contentFilter  func(file string) bool
	expectedStatus int
}

type validateCoordConfig struct {
	name           string
	x              string
	y              string
	expectedStatus int
}

type Link struct {
	A    xml.Name `xml:"a"`
	Href string   `xml:"href,attr"`
}

var server *httptest.Server

var runIntegrationTests = os.Getenv("RUN_IT") == "true"

var okUploadContent = &uploadTestConfig{
	manifest:     "test/data/contents.json",
	contentDir:   "test/data/demo",
	metadataPath: "test/data/metadata.json",
	contentFilter: func(file string) bool {
		return file[len(file)-1:] == "/"
	},
}

func TestMain(m *testing.M) {
	if runIntegrationTests {
		// Start server
		router := InitializeApp(config.GetConfig("config_test"))

		server = httptest.NewServer(router)
		defer server.Close()
		code := m.Run()

		os.Exit(code)
	}
}

func TestContentsHandlerS3Redirect(t *testing.T) {
	if !runIntegrationTests {
		t.Skip("Skipping integration test. To run it set RUN_IT=true")
	}
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

	assert.Equal(t, http.StatusMovedPermanently, response.StatusCode)

	link := new(Link)
	err = xml.NewDecoder(response.Body).Decode(link)
	if err != nil {
		t.Fatal("Error parsing response body")
	}

	c := config.GetConfig("config_test")

	expected := c.Storage.RemoteConfig.URL + CID

	assert.Equal(t, expected, link.Href, fmt.Sprintf("Should redirect to %s. Recieved link to : %s", expected, link.Href))

}

func TestInvalidCoordinates(t *testing.T) {
	if !runIntegrationTests {
		t.Skip("Skipping integration test. To run it set RUN_IT=true")
	}
	x1, y1, x2, y2 := 45, 45, 44, 46
	query := fmt.Sprintf("/mappings?nw=%d,%d&se=%d,%d", x1, y1, x2, y2)

	client := getNoRedirectClient()
	response, err := client.Get(server.URL + query)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	assert.Equal(t, http.StatusOK, response.StatusCode)

	contentType := response.Header.Get("Content-Type")
	assert.Equal(t, "application/json", contentType)

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Error(err)
	}
	bodyString := string(body)
	assert.Equal(t, "{}", bodyString)
}

func TestCoordinatesNotCached(t *testing.T) {
	if !runIntegrationTests {
		t.Skip("Skipping integration test. To run it set RUN_IT=true")
	}
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

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	bodyString := string(body)
	if bodyString != "{}" {
		t.Errorf("Mappings handler should return empty JSON when coordinates not in cache.\nRecieved:\n%s", bodyString)
	}
}

func TestUploadHandler(t *testing.T) {
	if !runIntegrationTests {
		t.Skip("Skipping integration test. To run it set RUN_IT=true")
	}
	response := execRequest(buildUploadRequest(okUploadContent, t), t)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("Upload unsuccessful. Got response code: %d", response.StatusCode)
	}
}

func TestGetContent(t *testing.T) {
	if !runIntegrationTests {
		t.Skip("Skipping integration test. To run it set RUN_IT=true")
	}
	rUpload := execRequest(buildUploadRequest(okUploadContent, t), t)
	assert.Equal(t, http.StatusOK, rUpload.StatusCode)

	client := server.Client()

	const testFileCID = "QmbdQuGbRFZdeqmK3PJyLV3m4p2KDELKRS4GfaXyehz672"
	resp, err := client.Get(fmt.Sprintf("%s/contents/%s", server.URL, testFileCID))
	if err != nil {
		t.Fatal(err)
	}

	if resp.StatusCode == http.StatusMovedPermanently {
		redirectURL := resp.Header.Get("Location")
		resp, err = client.Get(redirectURL)
		if err != nil {
			t.Fatal(err)
		}
	}

	testContents, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(testContents) != "something\n" {
		t.Errorf("Test file contents do not match.\nExpected 'something'\nGot %s", string(testContents))
	}
}

func TestValidateContent(t *testing.T) {
	if !runIntegrationTests {
		t.Skip("Skipping integration test. To run it set RUN_IT=true")
	}
	rUpload := execRequest(buildUploadRequest(okUploadContent, t), t)
	assert.Equal(t, http.StatusOK, rUpload.StatusCode)

	for _, tc := range validateTc {
		t.Run(tc.name, func(t *testing.T) {
			query := fmt.Sprintf("/validate?x=%s&y=%s", tc.x, tc.y)
			client := getNoRedirectClient()
			resp, err := client.Get(server.URL + query)
			if err != nil {
				t.Fatal()
			}
			assert.Equal(t, tc.expectedStatus, resp.StatusCode)
		})
	}
}

func TestContentStatus(t *testing.T) {
	if !runIntegrationTests {
		t.Skip("Skipping integration test. To run it set RUN_IT=true")
	}
	rUpload := execRequest(buildUploadRequest(okUploadContent, t), t)
	assert.Equal(t, http.StatusOK, rUpload.StatusCode)

	var contentsJSON []handlers.FileMetadata
	c, err := os.Open(okUploadContent.manifest)
	if err != nil {
		t.Fail()
	}
	defer c.Close()
	err = json.NewDecoder(c).Decode(&contentsJSON)
	if err != nil {
		t.Fail()
	}

	var list []string
	for _, content := range contentsJSON {
		if !strings.HasSuffix(content.Name, "/") {
			list = append(list, fmt.Sprintf("\"%s\"", content.Cid))
		}
	}

	list = append(list, "\"Not_A_CID\"")

	body := fmt.Sprintf("{\"content\": [%s]}", strings.Join(list, ","))

	req, err := http.NewRequest("POST", server.URL+"/content/status", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	if err != nil {
		t.Fail()
	}

	client := server.Client()
	response, err := client.Do(req)
	if err != nil {
		t.Fail()
	}
	defer response.Body.Close()
	assert.Equal(t, http.StatusOK, response.StatusCode)

	result := make(map[string]bool)
	err = json.NewDecoder(response.Body).Decode(&result)
	if err != nil {
		t.Fatal("Error parsing response body")
	}

	for k, v := range result {
		if k == "Not_A_CID" {
			assert.False(t, v)
		} else {
			assert.True(t, v)
		}
	}
}

func TestPartialUpload(t *testing.T) {
	if !runIntegrationTests {
		t.Skip("Skipping integration test. To run it set RUN_IT=true")
	}
	rUpload := execRequest(buildUploadRequest(okUploadContent, t), t)
	assert.Equal(t, http.StatusOK, rUpload.StatusCode)

	for _, tc := range redeployTC {
		t.Run(tc.name, func(t *testing.T) {
			rUpload := execRequest(buildUploadRequest(&tc, t), t)
			assert.Equal(t, tc.expectedStatus, rUpload.StatusCode)
		})
	}
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

func (conf *uploadTestConfig) readManifest() (*[]handlers.FileMetadata, error) {
	var manifest []handlers.FileMetadata
	c, err := os.Open(conf.manifest)
	if err != nil {
		return nil, err
	}
	defer c.Close()
	err = json.NewDecoder(c).Decode(&manifest)
	if err != nil {
		return nil, err
	}

	return &manifest, nil
}

func execRequest(r *http.Request, t *testing.T) *http.Response {
	client := server.Client()
	response, err := client.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func buildUploadRequest(config *uploadTestConfig, t *testing.T) *http.Request {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	err := loadUploadContent(config, writer)
	if err != nil {
		t.Fatal()
	}

	metadataFile := config.metadataPath
	var metadataBytes []byte
	metadataBytes, err = ioutil.ReadFile(metadataFile)
	if err != nil {
		t.Fatal()
	}
	var contentsBytes []byte
	contentsBytes, err = ioutil.ReadFile(config.manifest)
	if err != nil {
		t.Fatal()
	}

	_ = writer.WriteField("metadata", string(metadataBytes))
	rootCID := getRootCID(metadataFile)
	_ = writer.WriteField(rootCID, string(contentsBytes))

	err = writer.Close()
	if err != nil {
		t.Fatal()
	}

	req, err := http.NewRequest("POST", server.URL+"/mappings", body)
	if err != nil {
		t.Fatal()
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req
}

func loadUploadContent(c *uploadTestConfig, w *multipart.Writer) error {

	manifest, err := c.readManifest()
	if err != nil {
		return err
	}

	for _, content := range *manifest {
		if c.contentFilter(content.Name) {
			continue
		}

		part, err := w.CreateFormFile(content.Cid, content.Name)
		if err != nil {
			return err
		}

		dataPath := filepath.Join(c.contentDir, content.Name)
		var f *os.File
		f, err = os.Open(dataPath)
		if err != nil {
			log.Printf("Cannot open %s", dataPath)
			return err
		}
		_, err = io.Copy(part, f)
		if err != nil {
			return err
		}
	}
	return nil
}

func getNoRedirectClient() *http.Client {
	// Configure http.Client to avoid following redirects
	client := server.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return client
}

var validateTc = []validateCoordConfig{
	{
		name:           "Valid parcel",
		x:              "54",
		y:              "-136",
		expectedStatus: http.StatusOK,
	},
	{
		name:           "Invalid parcel",
		x:              "-10",
		y:              "10",
		expectedStatus: http.StatusNotFound,
	},
	{
		name:           "Invalid Coordinate",
		x:              "-10",
		y:              "s",
		expectedStatus: http.StatusNotFound,
	},
}

var redeployTC = []uploadTestConfig{
	{
		name:         "Full re deploy",
		manifest:     "test/data/contents.json",
		contentDir:   "test/data/demo",
		metadataPath: "test/data/metadata.json",
		contentFilter: func(file string) bool {
			return file[len(file)-1:] == "/"
		},
		expectedStatus: http.StatusOK,
	},
	{
		name:         "No New content",
		manifest:     "test/data/contents.json",
		contentDir:   "test/data/demo",
		metadataPath: "test/data/metadata.json",
		contentFilter: func(file string) bool {
			return file != "scene.json"
		},
		expectedStatus: http.StatusOK,
	},
	{
		name:         "Partial re deploy",
		manifest:     "test/data/contents.json",
		contentDir:   "test/data/demo",
		metadataPath: "test/data/metadata.json",
		contentFilter: func(file string) bool {
			return file != "scene.json" && file != "assets/test.txt"
		},
		expectedStatus: http.StatusOK,
	},
	{
		name:         "Missing content",
		manifest:     "test/data/missing-content.json",
		contentDir:   "test/data/demo",
		metadataPath: "test/data/metadata.json",
		contentFilter: func(file string) bool {
			return file[len(file)-1:] == "/" || file == "the-non-existing-asset.json"
		},
		expectedStatus: http.StatusBadRequest,
	},
}
