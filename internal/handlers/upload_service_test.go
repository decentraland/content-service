package handlers

import (
	"testing"

	"github.com/decentraland/content-service/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestValidateRequestSize(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()
	for _, tc := range sizeTestTable {
		mockStorage := mocks.NewMockStorage(mockController)
		for k, v := range tc.sizes {
			mockStorage.EXPECT().FileSize(k).Return(v, nil).AnyTimes()
		}
		us := &UploadServiceImpl{Storage: mockStorage, ParcelSizeLimit: tc.parcelMaxSize}

		err := us.validateRequestSize(tc.r)

		tc.errorsAssertion(t, err)
	}
}

type sizeCase struct {
	name            string
	parcelMaxSize   int64
	r               *UploadRequest
	sizes           map[string]int64
	errorsAssertion func(t assert.TestingT, object interface{}, msgAndArgs ...interface{}) bool
}

var sizeTestTable = []sizeCase{
	{
		name:          "Valid Size",
		parcelMaxSize: 1000,
		r: &UploadRequest{
			Scene:    &scene{Scene: sceneData{Parcels: []string{"0,0"}, Base: "0,0"}},
			Manifest: &[]FileMetadata{{Cid: "content", Name: "content"}},
		},
		sizes:           map[string]int64{"content": 1000},
		errorsAssertion: assert.Nil,
	}, {
		name:          "Valid Size - Multiple Elements",
		parcelMaxSize: 800,
		r: &UploadRequest{
			Scene: &scene{
				Scene: sceneData{Parcels: []string{"0,0"}, Base: "0,0"},
			},
			Manifest: &[]FileMetadata{{Cid: "content1", Name: "content1"}, {Cid: "content2", Name: "content2"}},
		},
		sizes:           map[string]int64{"content1": 400, "content2": 400},
		errorsAssertion: assert.Nil,
	}, {
		name:          "Invalid Size - Multiple Elements",
		parcelMaxSize: 800,
		r: &UploadRequest{
			Scene: &scene{
				Scene: sceneData{Parcels: []string{"0,0"}, Base: "0,0"},
			},
			Manifest: &[]FileMetadata{{Cid: "content1", Name: "content1"}, {Cid: "content2", Name: "content2"}},
		},
		sizes:           map[string]int64{"content1": 400, "content2": 410},
		errorsAssertion: assert.NotNil,
	}, {
		name:          "Valid Size - Multiple Elements and Parcels",
		parcelMaxSize: 800,
		r: &UploadRequest{
			Scene: &scene{
				Scene: sceneData{Parcels: []string{"0,0", "0,1"}, Base: "0,0"},
			},
			Manifest: &[]FileMetadata{{Cid: "content1", Name: "content1"}, {Cid: "content2", Name: "content2"}},
		},
		sizes:           map[string]int64{"content1": 400, "content2": 410},
		errorsAssertion: assert.Nil,
	},
}
