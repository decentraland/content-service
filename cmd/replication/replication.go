package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/structs"
	"github.com/go-redis/redis"
)

type parcelContent struct {
	ParcelID string            `json:"parcel_id"`
	Contents map[string]string `json:"contents"`
}

type metadata struct {
	Value        string `json:"value" structs:"value"`
	Signature    string `json:"signature" structs:"signature"`
	Validity     string `json:"validity" structs:"validity"`
	ValidityType string `json:"validityType" structs:"validityType"`
	Sequence     string `json:"sequence" structs:"sequence"`
	PubKey       string `json:"pubkey" structs:"pubkey"`
	RootCid      string `json:"-" structs:"rootcid"`
}

func main() {
	client := redis.NewClient(&redis.Options{
		Addr:     "content_service_redis:6379",
		Password: "",
		DB:       0,
	})

	url := "http://localhost:8000/mappings?nw=-150,150&se=150,-150"
	resp, err := http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var parcelContents []parcelContent
	err = json.NewDecoder(resp.Body).Decode(&parcelContents)
	if err != nil {
		panic(err)
	}

	for _, parcel := range parcelContents {
		xy := strings.Split(parcel.ParcelID, ",")
		validateURL := fmt.Sprintf("http://localhost:8000/validate?x=%s&y=%s", xy[0], xy[1])
		resp, err3 := http.Get(validateURL)
		if err3 != nil {
			panic(err3)
		}
		defer resp.Body.Close()

		var parcelMetadata *metadata
		err := json.NewDecoder(resp.Body).Decode(parcelMetadata)
		if err != nil {
			panic(err)
		}

		err = client.Set(parcel.ParcelID, parcelMetadata.RootCid, 0).Err()
		if err != nil {
			panic(err)
		}

		err = client.HMSet("metadata_"+parcelMetadata.RootCid, structs.Map(parcelMetadata)).Err()
		if err != nil {
			panic(err)
		}

		localPath := "backup/" + parcelMetadata.RootCid
		for filePath, cid := range parcel.Contents {
			err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
			if err != nil {
				panic(err)
			}

			file, err := os.Create(localPath + filePath)
			if err != nil {
				panic(err)
			}
			defer file.Close()

			contentsURL := "http://localhost:8000/contents?" + cid
			resp, err2 := http.Get(contentsURL)
			if err2 != nil {
				panic(err2)
			}
			defer resp.Body.Close()

			_, err3 := io.Copy(file, resp.Body)
			if err3 != nil {
				panic(err3)
			}

			err = client.HSet("content_"+parcelMetadata.RootCid, filePath, cid).Err()
			if err != nil {
				panic(err)
			}
		}
	}

}
