package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/decentraland/content-service/config"
	"github.com/decentraland/content-service/metrics"
	"github.com/decentraland/content-service/validation"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

const validRootCid = "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn"
const validSignature = "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b"
const validTestPubKey = "0xa08a656ac52c0b32902a76e122d2973b022caa0e"
const sceneJsonCID = "QmfRoY2437YZgrJK9s5Vvkj6z9xH4DqGT1VKp1WFoh6Ec4"

func TestRequestMetadataValidation(t *testing.T) {
	runValidationTests(metadataValidations, t)
}

func TestRequestFileMetadataValidation(t *testing.T) {
	runValidationTests(fileMetadataValidations, t)
}

func TestRequestSceneValidation(t *testing.T) {
	runValidationTests(sceneValidation, t)
}

func runValidationTests(cases []testDataValidation, t *testing.T) {
	v := validation.NewValidator()
	for _, tc := range cases {
		t.Run(tc.caseName, func(t *testing.T) {
			err := v.ValidateStruct(tc.s)
			tc.errorsAssertion(t, err)
		})
	}
}

type testDataValidation struct {
	caseName        string
	s               interface{}
	errorsAssertion func(t assert.TestingT, object interface{}, msgAndArgs ...interface{}) bool
}

var metadataValidations = []testDataValidation{
	{
		caseName: "Valid Metadata",
		s: Metadata{
			Value:        validRootCid,
			Signature:    validSignature,
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       validTestPubKey,
			RootCid:      validRootCid,
		},
		errorsAssertion: assert.Nil,
	}, {
		caseName: "Missing Root CID",
		s: Metadata{
			Value:        "",
			Signature:    validSignature,
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       validTestPubKey,
			RootCid:      "",
		},
		errorsAssertion: assert.NotNil,
	}, {
		caseName: "Missing Signature",
		s: Metadata{
			Value:        validRootCid,
			Signature:    "",
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       validTestPubKey,
			RootCid:      validRootCid,
		},
		errorsAssertion: assert.NotNil,
	}, {
		caseName: "Invalid Signature",
		s: Metadata{
			Value:        validRootCid,
			Signature:    "not a valid signature",
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       validTestPubKey,
			RootCid:      validRootCid,
		},
		errorsAssertion: assert.NotNil,
	}, {
		caseName: "Missing Key",
		s: Metadata{
			Value:        validRootCid,
			Signature:    validSignature,
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       "",
			RootCid:      validRootCid,
		},
		errorsAssertion: assert.NotNil,
	}, {
		caseName: "Invalid key",
		s: Metadata{
			Value:        validRootCid,
			Signature:    validSignature,
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       "Not the key you are looking for",
			RootCid:      validRootCid,
		},
		errorsAssertion: assert.NotNil,
	},
}

var fileMetadataValidations = []testDataValidation{
	{
		caseName: "Valid FileMetadata",
		s: FileMetadata{
			Cid:  sceneJsonCID,
			Name: "scene.json",
		},
		errorsAssertion: assert.Nil,
	},
	{
		caseName: "Missing CID",
		s: FileMetadata{
			Cid:  "",
			Name: "scene.json",
		},
		errorsAssertion: assert.NotNil,
	},
	{
		caseName: "Missing File Name",
		s: FileMetadata{
			Cid:  sceneJsonCID,
			Name: "",
		},
		errorsAssertion: assert.NotNil,
	},
}

var sceneValidation = []testDataValidation{
	{
		caseName: "Valid Scene file",
		s: scene{
			Display: display{
				Title: "suspicious_liskov",
			},
			Owner: validTestPubKey,
			Scene: sceneData{
				Parcels: []string{"54,-136"},
				Base:    "54,-136",
			},
			Communications: commsConfig{
				Type:       "webrtc",
				Signalling: "https://rendezvous.decentraland.org",
			},
			Main: "scene.js",
		},
		errorsAssertion: assert.Nil,
	}, {
		caseName: "Missing Parcels",
		s: scene{
			Display: display{
				Title: "suspicious_liskov",
			},
			Owner: validTestPubKey,
			Scene: sceneData{
				Parcels: []string{""},
				Base:    "",
			},
			Communications: commsConfig{
				Type:       "webrtc",
				Signalling: "https://rendezvous.decentraland.org",
			},
			Main: "scene.js",
		},
		errorsAssertion: assert.NotNil,
	},
}

func TestUploadRequestValidation(t *testing.T) {
	validator := validation.NewValidator()
	agent, _ := metrics.Make(config.Metrics{AppName: "", AppKey: "", AnalyticsKey: ""})

	for _, tc := range requestValidationTestCases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := buildUploadRequest(tc.cid, tc.scene, tc.sceneCid, tc.metadata, tc.content)
			if err != nil {
				t.Fatal(fmt.Scanf("Unexpected error: %s", err.Error()))
			}
			var filter *ContentTypeFilter
			if tc.filter == nil {
				filter = &ContentTypeFilter{filterPattern: ".*"}
			} else {
				filter = tc.filter
			}
			ctx := &UploadCtx{StructValidator: validator, Agent: agent, Filter: filter, Limits: config.Limits{ParcelAssetsLimit: tc.maxFiles}, TimeToLive: tc.ttl}
			request, err := ctx.parseRequest(r)
			tc.assert(t, request, err)
		})
	}

}

func buildUploadRequest(rootCID string, scene *scene, sceneCID string, metadata *Metadata, content *fileContent) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	manifest := []FileMetadata{
		{Cid: sceneCID, Name: "scene.json"},
	}

	if content != nil {
		manifest = append(manifest, *content.fm)
		part, err := writer.CreateFormFile(content.fm.Cid, content.fm.Name)
		if err != nil {
			return nil, err
		}

		_, err = io.Copy(part, strings.NewReader(content.content))
		if err != nil {
			return nil, err
		}
	}

	if metadata != nil {
		metaBytes, _ := json.Marshal(metadata)
		_ = writer.WriteField("metadata", string(metaBytes))
	}

	contentBytes, _ := json.Marshal(manifest)
	_ = writer.WriteField(rootCID, string(contentBytes))

	if scene != nil {
		sceneBytes, _ := json.Marshal(scene)
		part, err := writer.CreateFormFile(sceneCID, "scene.json")
		r := bytes.NewReader(sceneBytes)
		_, err = io.Copy(part, r)

		err = writer.Close()
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest("POST", "/mappings", body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, err
}

type requestValidation struct {
	name     string
	scene    *scene
	cid      string
	sceneCid string
	metadata *Metadata
	maxFiles int
	content  *fileContent
	filter   *ContentTypeFilter
	ttl      int64
	assert   func(t assert.TestingT, uploadRequest *UploadRequest, err error)
}

type fileContent struct {
	fm      *FileMetadata
	content string
}

func requestErrorAssertion(t assert.TestingT, uploadRequest *UploadRequest, err error) {
	assert.NotNil(t, err, "Expected error")
	assert.Nil(t, uploadRequest, "UploadRequest should be nil")
}

func requestAssertion(t assert.TestingT, uploadRequest *UploadRequest, err error) {
	assert.NotNil(t, uploadRequest, "UploadRequest should not be nil")
	assert.Nil(t, err, "Error should be nil")
}

var requestValidationTestCases = []requestValidation{
	{
		name: "Valid Request",
		scene: &scene{
			Display: display{
				Title: "suspicious_liskov",
			},
			Owner: validTestPubKey,
			Scene: sceneData{
				Parcels: []string{"54,-136"},
				Base:    "54,-136",
			},
			Communications: commsConfig{
				Type:       "webrtc",
				Signalling: "https://rendezvous.decentraland.org",
			},
			Main: "scene.js",
		},
		cid:      validRootCid,
		sceneCid: sceneJsonCID,
		metadata: &Metadata{
			Value:        validRootCid,
			Signature:    validSignature,
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       validTestPubKey,
			RootCid:      validRootCid,
			Timestamp:    time.Now().Unix(),
		},
		maxFiles: 1000,
		ttl:      600,
		assert:   requestAssertion,
	},
	{
		name: "Invalid Scene - Missing parcels",
		scene: &scene{
			Display: display{
				Title: "suspicious_liskov",
			},
			Owner: validTestPubKey,
			Scene: sceneData{},
			Communications: commsConfig{
				Type:       "webrtc",
				Signalling: "https://rendezvous.decentraland.org",
			},
			Main: "scene.js",
		},
		cid:      validRootCid,
		sceneCid: sceneJsonCID,
		metadata: &Metadata{
			Value:        validRootCid,
			Signature:    validSignature,
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       validTestPubKey,
			RootCid:      validRootCid,
			Timestamp:    time.Now().Unix(),
		},
		maxFiles: 1000,
		ttl:      600,
		assert:   requestErrorAssertion,
	}, {
		name:     "Missing Scene.json file",
		cid:      validRootCid,
		sceneCid: sceneJsonCID,
		metadata: &Metadata{
			Value:        validRootCid,
			Signature:    validSignature,
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       validTestPubKey,
			RootCid:      validRootCid,
			Timestamp:    time.Now().Unix(),
		},
		maxFiles: 1000,
		ttl:      600,
		assert:   requestErrorAssertion,
	}, {
		name: "Invalid Metadata - Missing Signaturet",
		scene: &scene{
			Display: display{
				Title: "suspicious_liskov",
			},
			Owner: validTestPubKey,
			Scene: sceneData{
				Parcels: []string{"54,-136"},
				Base:    "54,-136",
			},
			Communications: commsConfig{
				Type:       "webrtc",
				Signalling: "https://rendezvous.decentraland.org",
			},
			Main: "scene.js",
		},
		cid:      validRootCid,
		sceneCid: sceneJsonCID,
		metadata: &Metadata{
			Value:        validRootCid,
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       validTestPubKey,
			RootCid:      validRootCid,
			Timestamp:    time.Now().Unix(),
		},
		maxFiles: 1000,
		ttl:      600,
		assert:   requestErrorAssertion,
	}, {
		name: "Missing Metadata",
		scene: &scene{
			Display: display{
				Title: "suspicious_liskov",
			},
			Owner: validTestPubKey,
			Scene: sceneData{
				Parcels: []string{"54,-136"},
				Base:    "54,-136",
			},
			Communications: commsConfig{
				Type:       "webrtc",
				Signalling: "https://rendezvous.decentraland.org",
			},
			Main: "scene.js",
		},
		cid:      validRootCid,
		sceneCid: sceneJsonCID,
		maxFiles: 1000,
		ttl:      600,
		assert:   requestErrorAssertion,
	},
	{
		name: "Max files number exceeded",
		scene: &scene{
			Display: display{
				Title: "suspicious_liskov",
			},
			Owner: validTestPubKey,
			Scene: sceneData{
				Parcels: []string{"54,-136"},
				Base:    "54,-136",
			},
			Communications: commsConfig{
				Type:       "webrtc",
				Signalling: "https://rendezvous.decentraland.org",
			},
			Main: "scene.js",
		},
		metadata: &Metadata{
			Value:        validRootCid,
			Signature:    validSignature,
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       validTestPubKey,
			RootCid:      validRootCid,
			Timestamp:    time.Now().Unix(),
		},
		maxFiles: 1,
		cid:      validRootCid,
		sceneCid: sceneJsonCID,
		ttl:      600,
		assert:   requestErrorAssertion,
		content: &fileContent{
			fm: &FileMetadata{
				Cid:  uuid.New().String(),
				Name: "RandomFile",
			},
			content: uuid.New().String(),
		},
	},
	{
		name: "Filter Content Type",
		scene: &scene{
			Display: display{
				Title: "suspicious_liskov",
			},
			Owner: validTestPubKey,
			Scene: sceneData{
				Parcels: []string{"54,-136"},
				Base:    "54,-136",
			},
			Communications: commsConfig{
				Type:       "webrtc",
				Signalling: "https://rendezvous.decentraland.org",
			},
			Main: "scene.js",
		},
		cid:      validRootCid,
		sceneCid: sceneJsonCID,
		maxFiles: 1000,
		metadata: &Metadata{
			Value:        validRootCid,
			Signature:    validSignature,
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       validTestPubKey,
			RootCid:      validRootCid,
			Timestamp:    time.Now().Unix(),
		},
		content: &fileContent{
			fm: &FileMetadata{
				Cid:  uuid.New().String(),
				Name: "RandomFile",
			},
			content: uuid.New().String(),
		},
		filter: NewContentTypeFilter([]string{"application/javascript", "application/json"}),
		ttl:    600,
		assert: requestErrorAssertion,
	},
	{
		name: "Expired Request",
		scene: &scene{
			Display: display{
				Title: "suspicious_liskov",
			},
			Owner: validTestPubKey,
			Scene: sceneData{
				Parcels: []string{"54,-136"},
				Base:    "54,-136",
			},
			Communications: commsConfig{
				Type:       "webrtc",
				Signalling: "https://rendezvous.decentraland.org",
			},
			Main: "scene.js",
		},
		cid:      validRootCid,
		sceneCid: sceneJsonCID,
		metadata: &Metadata{
			Value:        validRootCid,
			Signature:    validSignature,
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       validTestPubKey,
			RootCid:      validRootCid,
			Timestamp:    time.Now().Unix() - 10000,
		},
		maxFiles: 1000,
		ttl:      1,
		assert:   requestErrorAssertion,
	},
}

type namingCase struct {
	name    string
	content *fileContent
}

func TestMultipartNaming(t *testing.T) {
	s := &scene{
		Display:        display{Title: "suspicious_liskov"},
		Owner:          validTestPubKey,
		Scene:          sceneData{Parcels: []string{"54,-136"}, Base: "54,-136"},
		Communications: commsConfig{Type: "webrtc", Signalling: "https://rendezvous.decentraland.org"},
		Main:           "scene.js",
	}

	m := &Metadata{
		Value:        validRootCid,
		Signature:    validSignature,
		Validity:     "2018-12-12T14:49:14.074000000Z",
		ValidityType: 0,
		Sequence:     2,
		PubKey:       validTestPubKey,
		RootCid:      validRootCid,
		Timestamp:    time.Now().Unix(),
	}

	dummyAgent, _ := metrics.Make(config.Metrics{AppName: "", AppKey: "", AnalyticsKey: ""})
	service := &uploadServiceMock{uploadedContent: make(map[string]string)}
	limits := config.Limits{ParcelSizeLimit: 150000, ParcelAssetsLimit: 1000}
	uploadCtx := UploadCtx{StructValidator: validation.NewValidator(), Service: service, Agent: dummyAgent, Filter: NewContentTypeFilter([]string{".*"}), Limits: limits, TimeToLive: 600}

	h := &ResponseHandler{Ctx: uploadCtx, H: UploadContent, Agent: dummyAgent, Id: "UploadContent"}

	for _, tc := range namingTestCases {
		t.Run(tc.name, func(t *testing.T) {
			request, err := buildUploadRequest(validRootCid, s, sceneJsonCID, m, tc.content)
			if err != nil {
				t.Fatal(fmt.Scanf("Unexpected error: %s", err.Error()))
			}
			rr := httptest.NewRecorder()
			h.ServeHTTP(rr, request)

			status := rr.Code

			assert.Equal(t, http.StatusOK, status)

			name, ok := service.uploadedContent[tc.content.fm.Cid]

			assert.True(t, ok)
			assert.Equal(t, tc.content.fm.Name, name)
		})
	}
}

var namingTestCases = []namingCase{
	{
		name: "Alphanumeric",
		content: &fileContent{
			fm: &FileMetadata{
				Cid:  uuid.New().String(),
				Name: "ABCDEFGHIJKLMNOPQRSTUVWabcdefghijklmnopqrstuv0123456789.txt",
			},
			content: uuid.New().String(),
		},
	},
	{
		name: "White Spaces",
		content: &fileContent{
			fm: &FileMetadata{
				Cid:  uuid.New().String(),
				Name: "one two three.txt",
			},
			content: uuid.New().String(),
		},
	}, {
		name: "Pound Char",
		content: &fileContent{
			fm: &FileMetadata{
				Cid:  uuid.New().String(),
				Name: "mambo#5.txt",
			},
			content: uuid.New().String(),
		},
	}, {
		name: "Compare Chars",
		content: &fileContent{
			fm: &FileMetadata{
				Cid:  uuid.New().String(),
				Name: "<mambo>.txt",
			},
			content: uuid.New().String(),
		},
	}, {
		name: "Percent Char",
		content: &fileContent{
			fm: &FileMetadata{
				Cid:  uuid.New().String(),
				Name: "%mambo%.txt",
			},
			content: uuid.New().String(),
		},
	}, {
		name: "Braces",
		content: &fileContent{
			fm: &FileMetadata{
				Cid:  uuid.New().String(),
				Name: "[{mambo}].txt",
			},
			content: uuid.New().String(),
		},
	}, {
		name: "Slashes",
		content: &fileContent{
			fm: &FileMetadata{
				Cid:  uuid.New().String(),
				Name: "\\|//.txt",
			},
			content: uuid.New().String(),
		},
	},
}

type uploadServiceMock struct {
	uploadedContent map[string]string
}

func (s *uploadServiceMock) ProcessUpload(r *UploadRequest) error {
	for k, v := range r.UploadedFiles {
		s.uploadedContent[k] = v[0].Filename
	}
	return nil
}

type filterCase struct {
	name           string
	filters        []string
	contentType    string
	expectedResult bool
}

func TestContentTypeFilter_FilterType(t *testing.T) {
	for _, tc := range filterTestCases {
		t.Run(tc.name, func(t *testing.T) {
			f := NewContentTypeFilter(tc.filters)
			assert.Equal(t, tc.expectedResult, f.IsAllowed(tc.contentType))
		})
	}
}

var filterTestCases = []filterCase{
	{
		name:           "IsAllowed Matching Content-type",
		filters:        []string{"application/octet-stream", "application/zip"},
		contentType:    "application/octet-stream",
		expectedResult: true,
	}, {
		name:           "FIler not Matching Content-type",
		filters:        []string{"application/octet-stream"},
		contentType:    "application/zip",
		expectedResult: false,
	}, {
		name:           "Allow based on regex",
		filters:        []string{"video.*"},
		contentType:    "video/mp4",
		expectedResult: true,
	}, {
		name:           "Allow everything - regex",
		filters:        []string{".*"},
		contentType:    "video/mp4",
		expectedResult: true,
	}, {
		name:           "IsAllowed everything - no filters",
		filters:        nil,
		contentType:    "video/mp4",
		expectedResult: true,
	},
}
