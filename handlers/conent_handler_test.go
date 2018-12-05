package handlers

import (
	"github.com/decentraland/content-service/validation"
	"github.com/stretchr/testify/assert"
	"net/http"
	"strings"
	"testing"
)

func TestContentStatusRequest(t *testing.T) {
	for _, tc := range requestData {
		t.Run(tc.name, func(t *testing.T) {
			var contentRequest ContentStatusRequest
			r, err := buildRequest(tc.jsonStr)
			if err != nil {
				t.Fail()
			}
			err = ExtractContentFormJsonRequest(r, &contentRequest, validation.NewValidator())
			tc.assertFunction(t, err, tc.errorMsg)
		})
	}
}

type contentStatusTestCase struct {
	name           string
	jsonStr        string
	assertFunction func(t *testing.T, err error, expectedError string)
	errorMsg       string
}

var requestData = []contentStatusTestCase{
	{
		name:           "OK",
		jsonStr:        "{\"content\": [\"QmQNuQ3qyJMe3vA2yhaS2fKPmJhXEKR3PcMHz3JZ5d6MSx\",\"QmdgsBkdXvsxaH8TTLLtxjFdfcbYJQL6UsZAwgDBumuq9C\"]}",
		assertFunction: assertOK,
	},
	{
		name:           "Empty Body",
		jsonStr:        "{}",
		assertFunction: assertError,
		errorMsg:       "Content field is required.",
	},
	{
		name:           "Invalid Type",
		jsonStr:        "{\"content\": [1]}",
		assertFunction: assertError,
		errorMsg:       "json: cannot unmarshal number",
	},
}

func assertOK(t *testing.T, err error, _ string) {
	assert.Nil(t, err)
}

func assertError(t *testing.T, err error, expectedError string) {
	assert.NotNil(t, err)
	assert.True(t, strings.Contains(err.Error(), expectedError))
}

func buildRequest(body string) (*http.Request, error) {
	req, err := http.NewRequest("POST", "http://content-service.org/content/status", strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	return req, err
}
