package data_test

import (
	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/mocks"
	"github.com/golang/mock/gomock"
	"strconv"
	"testing"
)

type userCanModifyParcelsTestData struct {
	inputKey       string
	inputParcel    string
	parcel         *data.Parcel
	expectedResult bool
	expectError    bool
	testCaseMsg    string
	resultMsg      string
	estate         *data.Estate
}

type isSignatureValidTestData struct {
	testCaseMsg    string
	inputMsg       string
	inputAddress   string
	inputSignature string
	expectedResult bool
	expectError    bool
	resultMsg      string
}

func TestUserCanModifyParcels(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()
	for _, d := range userCanModifyTable {
		t.Logf("[INPUT] = Given the publicKey %s and parcel [%s]", d.inputKey, d.inputParcel)

		mockDcl := mocks.NewMockDecentraland(mockController)
		mockDcl.EXPECT().GetParcel(d.parcel.X, d.parcel.Y).Return(d.parcel, nil).AnyTimes()
		if d.estate != nil {
			i, _ := strconv.Atoi(d.estate.ID)
			mockDcl.EXPECT().GetEstate(i).Return(d.estate, nil).AnyTimes()
		}

		t.Logf("[CASE] - %s", d.testCaseMsg)
		service := data.NewAuthorizationService(mockDcl)
		canModify, err := service.UserCanModifyParcels(d.inputKey, []string{d.inputParcel})

		validateResult(err, canModify, d.expectedResult, d.expectError, d.resultMsg, t)
	}
}

func TestIsSignatureValid(t *testing.T) {
	for _, d := range isSignatureValidTable {
		t.Logf("[INPUT] - Given the Message [%s], the Siganture [%s] and the Address [%s]", d.inputMsg, d.inputSignature, d.inputAddress)

		t.Logf("[CASE] - %s", d.testCaseMsg)
		service := data.NewAuthorizationService(data.NewDclClient(""))
		isValid, err := service.IsSignatureValid(d.inputMsg, d.inputSignature, d.inputAddress)

		validateResult(err, isValid, d.expectedResult, d.expectError, d.resultMsg, t)
	}
}

func validateResult(err error, result bool, expected bool, expectError bool, okMsg string, t *testing.T) {
	if err != nil && !expectError {
		t.Errorf("[x FAIL] - Function retrieve an Unexpected Error: %s", err.Error())
	}
	if err == nil && expectError {
		t.Error("[x FAIL] - Function should have retrieved an error")
	}
	if result != expected {
		t.Errorf("[x FAIL] - Function retrieved %t. Expected %t", result, expected)
	}
	if expectError {
		t.Logf("[✓ SUCCESS] - %s - Error: %s", okMsg, err.Error())
	} else {
		t.Logf("[✓ SUCCESS] - %s", okMsg)
	}
}

// UserCanModify Test cases to evaluate
var userCanModifyTable = []userCanModifyParcelsTestData{
	{
		testCaseMsg:    "When receiving an existing parcel with a valid owner",
		inputKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:    "1,2",
		parcel:         &data.Parcel{"id", 1, 2, "0xa08a656ac52c0b32902a76e122d2973b022caa0e", "", ""},
		expectedResult: true,
		expectError:    false,
		resultMsg:      "Then the user should be able to modify the parcel",
	}, {
		testCaseMsg:    "When the owner does not match the given key",
		inputKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:    "1,2",
		parcel:         &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "", ""},
		expectedResult: false,
		expectError:    false,
		resultMsg:      "Then the  user should not be able to modify the parcel",
	}, {
		testCaseMsg:    "When the owner does not match the given key, but it has Update operator privileges",
		inputKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:    "1,2",
		parcel:         &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "0xa08a656ac52c0b32902a76e122d2973b022caa0e", ""},
		expectedResult: true,
		expectError:    false,
		resultMsg:      "Then the  user should be able to modify the parcel",
	}, {
		testCaseMsg:    "When the input parcels are invalid",
		inputKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:    "not an integer,also not an integer",
		parcel:         &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "", ""},
		expectedResult: false,
		expectError:    true,
		resultMsg:      "Then the  operation should return an error",
	}, {
		testCaseMsg:    "When the user is estate Owner",
		inputKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:    "1,2",
		parcel:         &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "", "1"},
		expectedResult: true,
		expectError:    false,
		resultMsg:      "Then the user should be able to modify the parcel",
		estate: &data.Estate{ID: "1", Owner: "0xa08a656ac52c0b32902a76e122d2973b022caa0e", UpdateOperator: "", Data: struct {
			Parcels []*data.Parcel `json:"parcels"`
		}{Parcels: []*data.Parcel{}}},
	}, {
		testCaseMsg:    "When the user is not the Owner nor the estate Owner nor Update Operator",
		inputKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:    "1,2",
		parcel:         &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "", "1"},
		expectedResult: false,
		expectError:    false,
		resultMsg:      "Then the user should not be able to modify the parcel",
		estate: &data.Estate{ID: "1", Owner: "0x0000000000000000000000000000000000000000", UpdateOperator: "", Data: struct {
			Parcels []*data.Parcel `json:"parcels"`
		}{Parcels: []*data.Parcel{}}},
	}, {
		testCaseMsg:    "When the user is Estate Update operator",
		inputKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:    "1,2",
		parcel:         &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "", "1"},
		expectedResult: true,
		expectError:    false,
		resultMsg:      "Then the user should be able to modify the parcel",
		estate: &data.Estate{ID: "1", Owner: "0x0000000000000000000000000000000000000000", UpdateOperator: "0xa08a656ac52c0b32902a76e122d2973b022caa0e", Data: struct {
			Parcels []*data.Parcel `json:"parcels"`
		}{Parcels: []*data.Parcel{}}},
	},
}

var isSignatureValidTable = []isSignatureValidTestData{
	{
		testCaseMsg:    "When validating the  signature  with the correct messgage and signature",
		inputMsg:       "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		inputAddress:   "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputSignature: "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b",
		expectedResult: true,
		expectError:    false,
		resultMsg:      "Then the result should be true",
	}, {
		testCaseMsg:    "When validating the signature with the not corresponding message",
		inputMsg:       "not the message you signed",
		inputAddress:   "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputSignature: "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b",
		expectedResult: false,
		expectError:    false,
		resultMsg:      "Then the result should be false",
	}, {
		testCaseMsg:    "When validating a different signature with not corresponding address",
		inputMsg:       "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		inputAddress:   "0x0000000000000000000000000000000000000000",
		inputSignature: "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b",
		expectedResult: false,
		expectError:    false,
		resultMsg:      "Then the result should be false",
	}, {
		testCaseMsg:    "When validating the invalid signature",
		inputMsg:       "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		inputAddress:   "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputSignature: "not hex, not a signtature",
		expectedResult: false,
		expectError:    true,
		resultMsg:      "Then the result should be an error",
	}, {
		testCaseMsg:    "When validating the signature with the invalid address",
		inputMsg:       "QmeoVuRM2ynxMfBn6eEqeTVRkJR9KZBQbLMLakZjioNhdn",
		inputAddress:   "not an address",
		inputSignature: "0x96a6e3f69b25fcf89d5af9fb9d6f17da8dd86548f486822e74296af1d8bcaf920e67684e2a15cd942526a4ede10dd5483eccb381d92f88b932858d7a466f99ed1b",
		expectedResult: false,
		expectError:    true,
		resultMsg:      "Then the result should be an error",
	},
}
