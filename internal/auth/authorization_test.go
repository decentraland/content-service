package auth_test

import (
	"strconv"
	"strings"
	"testing"

	"github.com/decentraland/content-service/internal/decentraland"

	"github.com/decentraland/content-service/internal/auth"
	"github.com/decentraland/content-service/internal/metrics"
	"github.com/decentraland/content-service/mocks"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type userCanModifyParcelsTestData struct {
	inputKey     string
	inputParcel  string
	accessData   *decentraland.AccessData
	testCaseName string
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

func TestUserCanModifyParcels(t *testing.T) {
	mockController := gomock.NewController(t)
	defer mockController.Finish()
	for _, tc := range userCanModifyTable {
		t.Run(tc.testCaseName, func(t *testing.T) {
			mockDcl := mocks.NewMockDecentraland(mockController)
			if tc.accessData != nil {
				coords := strings.Split(tc.accessData.Id, ",")
				x, _ := strconv.Atoi(coords[0])
				y, _ := strconv.Atoi(coords[1])
				mockDcl.EXPECT().GetParcelAccessData(tc.inputKey, int64(x), int64(y)).Return(tc.accessData, nil).AnyTimes()
			}
			service := auth.NewAuthorizationService(mockDcl)
			canModify, err := service.UserCanModifyParcels(tc.inputKey, []string{tc.inputParcel})
			tc.evalResult(err, canModify, t)
		})
	}
}

func TestIsSignatureValid(t *testing.T) {
	a, _ := metrics.Make(metrics.Config{AppName: "", Enabled: false, AnalyticsKey: ""})
	for _, tc := range isSignatureValidTable {
		t.Run(tc.testCaseName, func(t *testing.T) {
			service := auth.NewAuthorizationService(decentraland.NewDclClient("", a))
			isValid := service.IsSignatureValid(tc.inputMsg, tc.inputSignature, tc.inputAddress)
			tc.evalResult(t, isValid)
		})
	}
}

// UserCanModify Test cases to evaluate
var userCanModifyTable = []userCanModifyParcelsTestData{
	{
		testCaseName: "Address Approved",
		inputKey:     "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:  "1,2",
		accessData:   &decentraland.AccessData{Id: "1,2", Address: "0xa08a656ac52c0b32902a76e122d2973b022caa0e", IsUpdateAuthorized: true},
		evalResult:   expectTrue,
	},
	{
		testCaseName: "Address Unauthorized",
		inputKey:     "0xa08a656ac52c0b32902a76e122d2973b022caa0e",
		inputParcel:  "1,2",
		accessData:   &decentraland.AccessData{Id: "1,2", Address: "0xa08a656ac52c0b32902a76e122d2973b022caa0e", IsUpdateAuthorized: false},
		evalResult:   expectFalse,
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
