package data_test

import (
	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"strconv"
	"testing"
)

type userCanModifyParcelsTestData struct {
	inputKey     string
	inputParcel  string
	parcel       *data.Parcel
	testCaseName string
	estate       *data.Estate
	evalResult   evalBooleanResult
}

type isSignatureValidTestData struct {
	testCaseName   string
	inputMsg       string
	inputAddress   string
	inputSignature string
	evalResult     func(t assert.TestingT, value bool, msgAndArgs ...interface{}) bool
}

type evalBooleanResult func(err error, value bool, t *testing.T)

func expectTrue(err error, value bool, t *testing.T) {
	assert.Nil(t, err)
	assert.True(t, value)
}

func expectFalse(err error, value bool, t *testing.T) {
	assert.Nil(t, err)
	assert.False(t, value)
}

func expectError(err error, value bool, t *testing.T) {
	assert.NotNil(t, err)
}

func TestUserCanModifyParcels(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()
	for _, tc := range userCanModifyTable {
		t.Run(tc.testCaseName, func(t *testing.T) {
			mockDcl := mocks.NewMockDecentraland(mockController)
			mockDcl.EXPECT().GetParcel(tc.parcel.X, tc.parcel.Y).Return(tc.parcel, nil).AnyTimes()
			if tc.estate != nil {
				i, _ := strconv.Atoi(tc.estate.ID)
				mockDcl.EXPECT().GetEstate(i).Return(tc.estate, nil).AnyTimes()
			}
			service := data.NewAuthorizationService(mockDcl)
			canModify, err := service.UserCanModifyParcels(tc.inputKey, []string{tc.inputParcel})
			tc.evalResult(err, canModify, t)
		})
	}
}

func TestIsSignatureValid(t *testing.T) {
	for _, tc := range isSignatureValidTable {
		t.Run(tc.testCaseName, func(t *testing.T) {
			service := data.NewAuthorizationService(data.NewDclClient(""))
			isValid := service.IsSignatureValid(tc.inputMsg, tc.inputSignature, tc.inputAddress)
			tc.evalResult(t, isValid)
		})
	}
}

// UserCanModify Test cases to evaluate
var userCanModifyTable = []userCanModifyParcelsTestData{
	{
		testCaseName: "Parcel with a valid owner",
		inputKey:     "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:  "1,2",
		parcel:       &data.Parcel{"id", 1, 2, "0xa08a656ac52c0b32902a76e122d2973b022caa0e", "", ""},
		evalResult:   expectTrue,
	}, {
		testCaseName: "Owner does not match the given key",
		inputKey:     "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:  "1,2",
		parcel:       &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "", ""},
		evalResult:   expectFalse,
	}, {
		testCaseName: "Owner does not match the given key, but it has Update operator privileges",
		inputKey:     "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:  "1,2",
		parcel:       &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "0xa08a656ac52c0b32902a76e122d2973b022caa0e", ""},
		evalResult:   expectTrue,
	}, {
		testCaseName: "Input parcels are invalid",
		inputKey:     "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:  "not an integer,also not an integer",
		parcel:       &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "", ""},
		evalResult:   expectError,
	}, {
		testCaseName: "The user is estate Owner",
		inputKey:     "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:  "1,2",
		parcel:       &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "", "1"},
		evalResult:   expectTrue,
		estate: &data.Estate{ID: "1", Owner: "0xa08a656ac52c0b32902a76e122d2973b022caa0e", UpdateOperator: "", Data: struct {
			Parcels []*data.Parcel `json:"parcels"`
		}{Parcels: []*data.Parcel{}}},
	}, {
		testCaseName: "The user is not the Owner nor the estate Owner nor Update Operator",
		inputKey:     "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:  "1,2",
		parcel:       &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "", "1"},
		evalResult:   expectFalse,
		estate: &data.Estate{ID: "1", Owner: "0x0000000000000000000000000000000000000000", UpdateOperator: "", Data: struct {
			Parcels []*data.Parcel `json:"parcels"`
		}{Parcels: []*data.Parcel{}}},
	}, {
		testCaseName: "User is Estate Update operator",
		inputKey:     "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:  "1,2",
		parcel:       &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "", "1"},
		evalResult:   expectTrue,
		estate: &data.Estate{ID: "1", Owner: "0x0000000000000000000000000000000000000000", UpdateOperator: "0xa08a656ac52c0b32902a76e122d2973b022caa0e", Data: struct {
			Parcels []*data.Parcel `json:"parcels"`
		}{Parcels: []*data.Parcel{}}},
	},
}

var isSignatureValidTable = []isSignatureValidTestData{
	{
		testCaseName:   "Validate the signature with the correct messgage and address",
		inputMsg:       "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		inputAddress:   "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputSignature: "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b",
		evalResult:     assert.True,
	}, {
		testCaseName:   "Validate signature with the not corresponding message",
		inputMsg:       "not the message you signed",
		inputAddress:   "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputSignature: "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b",
		evalResult:     assert.False,
	}, {
		testCaseName:   "Validate signature with not the corresponding address",
		inputMsg:       "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		inputAddress:   "0x0000000000000000000000000000000000000000",
		inputSignature: "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b",
		evalResult:     assert.False,
	}, {
		testCaseName:   "Invalid signature",
		inputMsg:       "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		inputAddress:   "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputSignature: "not hex, not a signtature",
		evalResult:     assert.False,
	}, {
		testCaseName:   "Invalid address",
		inputMsg:       "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		inputAddress:   "not an address",
		inputSignature: "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b",
		evalResult:     assert.False,
	},
}
