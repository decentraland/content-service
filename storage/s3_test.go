package storage

import (
	"testing"

	"github.com/magiconair/properties/assert"
)

func TestGetFile(t *testing.T) {
	for _, tc := range getFileTestCases {
		t.Run(tc.name, func(t *testing.T) {
			output := tc.s3.GetFile(tc.cid)
			assert.Equal(t, output, tc.result)
		})
	}
}

type getFileData struct {
	name   string
	s3     S3
	cid    string
	result string
}

var getFileTestCases = []getFileData{
	{
		name: "Base url without a '/' at the end",
		s3: S3{
			URL: "https://s3bucket.com",
		},
		cid:    "theCid",
		result: "https://s3bucket.com/theCid",
	}, {
		name: "Base url with a '/' at the end",
		s3: S3{
			URL: "https://s3bucket.com/",
		},
		cid:    "theCid",
		result: "https://s3bucket.com/theCid",
	},
}
