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

// Test cases to evaluate
var userCanModifyTable = []userCanModifyParcelsTestData{
	{testCaseMsg: "When receiving an existing parcel with a valid owner",
		inputKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:    "1,2",
		parcel:         &data.Parcel{"id", 1, 2, "0xa08a656ac52c0b32902a76e122d2973b022caa0e", "", ""},
		expectedResult: true,
		expectError:    false,
		resultMsg:      "Then the user should be able to modify the parcel"},

	{testCaseMsg: "When the owner does not match the given key",
		inputKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:    "1,2",
		parcel:         &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "", ""},
		expectedResult: false,
		expectError:    false,
		resultMsg:      "Then the  user should not be able to modify the parcel"},

	{testCaseMsg: "When the owner does not match the given key, but it has Update operator privileges",
		inputKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:    "1,2",
		parcel:         &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "0xa08a656ac52c0b32902a76e122d2973b022caa0e", ""},
		expectedResult: true,
		expectError:    false,
		resultMsg:      "Then the  user should be able to modify the parcel"},

	{testCaseMsg: "When the input parcels are invalid",
		inputKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:    "not an integer,also not an integer",
		parcel:         &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "", ""},
		expectedResult: false,
		expectError:    true,
		resultMsg:      "Then the  operation should return an error"},

	{testCaseMsg: "When the user is estate Owner",
		inputKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:    "1,2",
		parcel:         &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "", "1"},
		expectedResult: true,
		expectError:    false,
		resultMsg:      "Then the user should be able to modify the parcel",
		estate: &data.Estate{ID: "1", Owner: "0xa08a656ac52c0b32902a76e122d2973b022caa0e", UpdateOperator: "", Data: struct {
			Parcels []*data.Parcel `json:"parcels"`
		}{Parcels: []*data.Parcel{}}},
	},

	{testCaseMsg: "When the user is not the Owner nor the estate Owner nor Update Operator",
		inputKey:       "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:    "1,2",
		parcel:         &data.Parcel{"id", 1, 2, "0x0000000000000000000000000000000000000000", "", "1"},
		expectedResult: false,
		expectError:    false,
		resultMsg:      "Then the user should not be able to modify the parcel",
		estate: &data.Estate{ID: "1", Owner: "0x0000000000000000000000000000000000000000", UpdateOperator: "", Data: struct {
			Parcels []*data.Parcel `json:"parcels"`
		}{Parcels: []*data.Parcel{}}},
	},

	{testCaseMsg: "When the user is Estate Update operator",
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

func TestUserCanModifyParcels(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()
	for _, d := range userCanModifyTable {
		t.Logf("Given the publicKey %s and parcel [%s]", d.inputKey, d.inputParcel)

		mockDcl := mocks.NewMockDecentraland(mockController)
		mockDcl.EXPECT().GetParcel(d.parcel.X, d.parcel.Y).Return(d.parcel, nil).AnyTimes()
		if d.estate != nil {
			i, _ := strconv.Atoi(d.estate.ID)
			mockDcl.EXPECT().GetEstate(i).Return(d.estate, nil).AnyTimes()
		}

		t.Log(d.testCaseMsg)
		service := data.NewAuthorizationService(mockDcl)
		canModify, err := service.UserCanModifyParcels(d.inputKey, []string{d.inputParcel})

		if err != nil && !d.expectError {
			t.Errorf("[FAIL] - Function retrieve an Unexpected Error: %s", err.Error())
		}
		if canModify != d.expectedResult {
			t.Errorf("[FAIL] - Function retrieved %t. Expected %t", canModify, d.expectedResult)
		}
		t.Logf("[SUCCESS] - %s", d.resultMsg)
	}
}
