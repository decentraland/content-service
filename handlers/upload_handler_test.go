package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/decentraland/content-service/metrics"
	"github.com/decentraland/content-service/validation"
	"github.com/stretchr/testify/assert"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
)

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
			Value:        "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
			Signature:    "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b",
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
			RootCid:      "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		},
		errorsAssertion: assert.Nil,
	}, {
		caseName: "Missing Root CID",
		s: Metadata{
			Value:        "",
			Signature:    "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b",
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
			RootCid:      "",
		},
		errorsAssertion: assert.NotNil,
	}, {
		caseName: "Missing Signature",
		s: Metadata{
			Value:        "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
			Signature:    "",
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
			RootCid:      "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		},
		errorsAssertion: assert.NotNil,
	}, {
		caseName: "Invalid Signature",
		s: Metadata{
			Value:        "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
			Signature:    "not a valid signature",
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
			RootCid:      "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		},
		errorsAssertion: assert.NotNil,
	}, {
		caseName: "Missing Key",
		s: Metadata{
			Value:        "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
			Signature:    "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b",
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       "",
			RootCid:      "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		},
		errorsAssertion: assert.NotNil,
	}, {
		caseName: "Invalid key",
		s: Metadata{
			Value:        "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
			Signature:    "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b",
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       "Not the key you are looking for",
			RootCid:      "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		},
		errorsAssertion: assert.NotNil,
	},
}

var fileMetadataValidations = []testDataValidation{
	{
		caseName: "Valid FileMetadata",
		s: FileMetadata{
			Cid:  "QmfRoY2437YZgrJK9s5Vvkj6z9xH4DqGT1VKp1WFoh6Ec4",
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
			Cid:  "QmfRoY2437YZgrJK9s5Vvkj6z9xH4DqGT1VKp1WFoh6Ec4",
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
			Owner: "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
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
			Owner: "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
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
	agent, _ := metrics.Make("", "")

	for _, tc := range requestValidationTestCases {
		t.Run(tc.name, func(t *testing.T) {
			r, err := buildUploadRequest(tc.cid, tc.scene, tc.sceneCid, tc.metadata)
			if err != nil {
				t.Fatal(fmt.Scanf("Unexpected error: %s", err.Error()))
			}
			request, err := parseRequest(r, validator, agent)
			tc.assert(t, request, err)
		})
	}

}

func buildUploadRequest(rootCID string, scene *scene, sceneCID string, metadata *Metadata) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	manifest := []FileMetadata{
		{Cid: sceneCID, Name: "scene.json"},
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
	assert   func(t assert.TestingT, uploadRequest *UploadRequest, err error)
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
			Owner: "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
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
		cid:      "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		sceneCid: "QmfRoY2437YZgrJK9s5Vvkj6z9xH4DqGT1VKp1WFoh6Ec4",
		metadata: &Metadata{
			Value:        "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
			Signature:    "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b",
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
			RootCid:      "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		},
		assert: requestAssertion,
	}, {
		name: "Invalid Scene - Missing parcels",
		scene: &scene{
			Display: display{
				Title: "suspicious_liskov",
			},
			Owner: "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
			Scene: sceneData{},
			Communications: commsConfig{
				Type:       "webrtc",
				Signalling: "https://rendezvous.decentraland.org",
			},
			Main: "scene.js",
		},
		cid:      "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		sceneCid: "QmfRoY2437YZgrJK9s5Vvkj6z9xH4DqGT1VKp1WFoh6Ec4",
		metadata: &Metadata{
			Value:        "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
			Signature:    "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b",
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
			RootCid:      "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		},
		assert: requestErrorAssertion,
	}, {
		name:     "Missing Scene.json file",
		cid:      "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		sceneCid: "QmfRoY2437YZgrJK9s5Vvkj6z9xH4DqGT1VKp1WFoh6Ec4",
		metadata: &Metadata{
			Value:        "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
			Signature:    "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b",
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
			RootCid:      "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		},
		assert: requestErrorAssertion,
	}, {
		name: "Invalid Metadata - Missing Signaturet",
		scene: &scene{
			Display: display{
				Title: "suspicious_liskov",
			},
			Owner: "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
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
		cid:      "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		sceneCid: "QmfRoY2437YZgrJK9s5Vvkj6z9xH4DqGT1VKp1WFoh6Ec4",
		metadata: &Metadata{
			Value:        "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
			Validity:     "2018-12-12T14:49:14.074000000Z",
			ValidityType: 0,
			Sequence:     2,
			PubKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
			RootCid:      "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		},
		assert: requestErrorAssertion,
	}, {
		name: "Missing Metadata",
		scene: &scene{
			Display: display{
				Title: "suspicious_liskov",
			},
			Owner: "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
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
		cid:      "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		sceneCid: "QmfRoY2437YZgrJK9s5Vvkj6z9xH4DqGT1VKp1WFoh6Ec4",
		assert:   requestErrorAssertion,
	},
}
