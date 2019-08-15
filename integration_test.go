package main

import (
	"bytes"
	"crypto/ecdsa"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"
	ginlogrus "github.com/toorop/gin-logrus"

	"github.com/decentraland/content-service/test/utils"
	"github.com/ethereum/go-ethereum/common/hexutil"

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

	"github.com/stretchr/testify/assert"

	"github.com/decentraland/content-service/config"
	"github.com/decentraland/content-service/internal/handlers"
	"github.com/ethereum/go-ethereum/crypto"
)

type uploadTestConfig struct {
	name           string
	metadataPath   string
	contentDir     string
	manifest       string
	contentFilter  func(file string) bool
	expectedStatus int
	extraContent   func() *utils.FileMetadata
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
	extraContent: nil,
}

type ParcelContent struct {
	ParcelID  string            `json:"parcel_id"`
	Contents  []*ContentElement `json:"contents"`
	RootCID   string            `json:"root_cid"`
	Publisher string            `json:"publisher"`
}

type MappedSceneContent struct {
	Data []*SceneContent `json:"data"`
}

type SceneContent struct {
	SceneCID string         `json:"scene_cid"`
	RootCID  string         `json:"root_cid"`
	Content  *ParcelContent `json:"content"`
}

type MappedScenes struct {
	Data []*Scene `json:"data"`
}

type Scene struct {
	ParcelId string `json:"parcel_id"`
	SceneCID string `json:"scene_cid"`
	RootCID  string `json:"root_cid"`
}

type ContentElement struct {
	File string `json:"file"`
	Cid  string `json:"hash"`
}

var scenesUploadContent = &uploadTestConfig{
	manifest:     "test/data3/contents.json",
	contentDir:   "test/data3/upload",
	metadataPath: "test/data3/metadata.json",
	contentFilter: func(file string) bool {
		return file[len(file)-1:] == "/"
	},
	extraContent: nil,
}

var scenesUploadContent2 = &uploadTestConfig{
	manifest:     "test/data4/contents.json",
	contentDir:   "test/data4/upload",
	metadataPath: "test/data4/metadata.json",
	contentFilter: func(file string) bool {
		return file[len(file)-1:] == "/"
	},
	extraContent: nil,
}

func TestMain(m *testing.M) {
	if runIntegrationTests {
		conf := config.GetConfig("config_test")
		l := newLogger()
		l.SetLevel(logrus.PanicLevel)
		router := gin.New()
		router.Use(ginlogrus.Logger(l), gin.Recovery())
		// Start server
		InitializeHandler(router, conf, l)
		server = httptest.NewServer(router)
		defer server.Close()
		code := m.Run()

		os.Exit(code)
	}
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

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Error(err)
	}
	bodyString := string(body)
	assert.Equal(t, "[]", bodyString)
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

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	bodyString := string(body)
	if bodyString != "[]" {
		t.Errorf("Mappings handler should return empty JSON List when coordinates not in cache.\nRecieved:\n%s", bodyString)
	}
}

func TestScenes(t *testing.T) {
	if !runIntegrationTests {
		t.Skip("Skipping integration test. To run it set RUN_IT=true")
	}

	response := execRequest(buildUploadRequest(scenesUploadContent, t), t)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("Upload unsuccessful. Got response code: %d", response.StatusCode)
	}

	x1, y1, x2, y2 := 143, -93, 143, -93
	query := fmt.Sprintf("/scenes?x1=%d&y1=%d&x2=%d&y2=%d", x1, y1, x2, y2)

	client := getNoRedirectClient()
	response, err := client.Get(server.URL + query)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Error("Mappings handler should respond with status code 200. Recieved code: ", response.StatusCode)
	}

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	var cids MappedScenes
	err = json.Unmarshal(body, &cids)
	if err != nil {
		t.Errorf("Wrong json")
	}

	oldCid := ""
	for _, p := range cids.Data {
		if p.ParcelId != "143,-93" {
			oldCid = p.RootCID
			break
		}
	}
	if oldCid == "" {
		t.Errorf("Parcel not found")
	}

	response = execRequest(buildUploadRequest(scenesUploadContent2, t), t)
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Fatalf("Upload unsuccessful. Got response code: %d", response.StatusCode)
	}

	x1, y1, x2, y2 = 143, -93, 144, -93
	query = fmt.Sprintf("/scenes?x1=%d&y1=%d&x2=%d&y2=%d", x1, y1, x2, y2)

	response, err = client.Get(server.URL + query)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Error("Mappings handler should respond with status code 200. Recieved code: ", response.StatusCode)
	}

	body, err = ioutil.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal([]byte(body), &cids)
	if err != nil {
		t.Errorf("Wrong json")
	}

	parcelA := ""
	parcelB := ""
	for _, p := range cids.Data {
		if p.ParcelId == "143,-93" {
			parcelA = p.RootCID
		}
		if p.ParcelId == "144,-93" {
			parcelB = p.RootCID
		}
	}
	if parcelA != "" {
		t.Errorf("Parcel A must be invalid now")
	}
	if parcelB == oldCid {
		t.Errorf("Cid didn't get updated")
	}

	///////////////////////////////////////////////////////////////////////////////
	query = fmt.Sprintf("/parcel_info?cids=%s,%s", parcelB, parcelA)

	response, err = client.Get(server.URL + query)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err = ioutil.ReadAll(response.Body)

	var content MappedSceneContent
	err = json.Unmarshal(body, &content)
	if err != nil {
		t.Errorf("Can't parse parcel_info response")
	}
	if len(content.Data) != 1 {
		t.Errorf("Found more answers than expected")
	}

	if content.Data[0].RootCID != parcelB {
		t.Errorf("Should find metadata for scene %s", parcelB)
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

func TestUploadHandler2(t *testing.T) {
	if !runIntegrationTests {
		t.Skip("Skipping integration test. To run it set RUN_IT=true")
	}
	response := execRequest(buildUploadRequest(scenesUploadContent2, t), t)
	defer response.Body.Close()

	resp, _ := ioutil.ReadAll(response.Body)
	if response.StatusCode != http.StatusOK {
		t.Fatalf("Upload unsuccessful. Got response code: %d", response.StatusCode)
		t.Fatalf("Need to use this acr %T", resp)
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

func getPrivateKey() (*ecdsa.PrivateKey, string) {
	privateKey := os.Getenv("TEST_PRIVATEKEY")
	pkbytes, _ := hexutil.Decode(privateKey)
	key, _ := crypto.ToECDSA(pkbytes)
	return key, os.Getenv("TEST_ADDRESS")
}

func signRootCid(cid string, timestamp int64, key *ecdsa.PrivateKey) []byte {
	msg := cid + "." + fmt.Sprintf("%d", timestamp)
	msg = fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(msg), msg)
	hash := crypto.Keccak256Hash([]byte(msg))
	sig, _ := crypto.Sign(hash.Bytes(), key)
	return sig
}

func buildUploadRequest(config *uploadTestConfig, t *testing.T) *http.Request {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	ipfsNode, _ := utils.InitIpfsNode()
	filesCids, _ := utils.ToFileData(config.contentDir, ipfsNode)
	key, address := getPrivateKey()

	if config.extraContent != nil {
		filesCids = append(filesCids, config.extraContent())
	}

	contentjson, _ := json.Marshal(filesCids)

	err := loadUploadContent(config, writer, filesCids)
	if err != nil {
		t.Fatal()
	}

	now := time.Now().Unix()
	rootCID, _ := utils.CalculateRootCid(config.contentDir, ipfsNode)

	sig := signRootCid(rootCID, now, key)

	metadata := &handlers.Metadata{
		PubKey:       address,
		Value:        rootCID,
		RootCid:      rootCID,
		Signature:    hexutil.Encode(sig),
		Timestamp:    now,
		Validity:     "2018-12-12T14:49:14.074000000Z",
		ValidityType: 0,
		Sequence:     2,
	}

	mbytes, _ := json.Marshal(metadata)
	_ = writer.WriteField("metadata", string(mbytes))
	_ = writer.WriteField(metadata.RootCid, string(contentjson))

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

func loadUploadContent(c *uploadTestConfig, w *multipart.Writer, cids []*utils.FileMetadata) error {

	manifest, err := c.readManifest()
	if err != nil {
		return err
	}

	for _, content := range *manifest {
		if c.contentFilter(content.Name) {
			continue
		}

		cid := content.Cid
		for _, meta := range cids {
			if meta.Name[1:len(meta.Name)] == content.Name {
				cid = meta.Cid
				break
			}
		}
		part, err := w.CreateFormFile(cid, content.Name)
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
		x:              "65",
		y:              "-135",
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
		expectedStatus: http.StatusBadRequest,
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
		extraContent: func() *utils.FileMetadata {
			return &utils.FileMetadata{Cid: "clearlynotcid", Name: "the-non-existing-asset.json"}
		},
	},
}
