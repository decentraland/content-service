package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"flag"

	"github.com/decentraland/content-service/config"
	"github.com/decentraland/content-service/handlers"
	"github.com/decentraland/content-service/storage"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/fatih/structs"
	"github.com/go-redis/redis"
	"github.com/spf13/viper"
)

type parcelContent struct {
	ParcelID string            `json:"parcel_id"`
	Contents map[string]string `json:"contents"`
}

var conf *config.Configuration
var client *redis.Client

func init() {
	conf := config.GetConfig("config")
	
	client = redis.NewClient(&redis.Options{
		Addr:     conf.Redis.Address,
		Password: conf.Redis.Password,
		DB:       conf.Redis.DB,
	})
}

func main() {
	args := flag.Args()
	if len(args) != 4 {
		log.Fatal("Please provide four mapping coordinates.\n\nUsage: ./replication nw1 nw2 se1 se2")
	}
	serverURL := getServerURL(conf.Server.URL, conf.Server.Port)
	mappingsURL := fmt.Sprintf("%s/mappings?nw=%s,%s&se=%s,%s", serverURL, args[0], args[1], args[2], args[3])
	resp, err := http.Get(mappingsURL)
	if err != nil {
		log.Fatalf("Failed to get url %s", mappingsURL)
	}
	defer resp.Body.Close()

	var store *storage.Storage
	if conf.S3Storage.Bucket == "" {
		storage = storage.NewS3(config.S3Storage.Bucket, config.S3Storage.ACL, config.S3Storage.URL)
	} else {
		storage = storage.NewLocal(config.LocalStorage)
	}

	var parcelContents []parcelContent
	err = json.NewDecoder(resp.Body).Decode(&parcelContents)
	if err != nil {
		log.Fatal("Cannot parse response\n", err)
	}

	for _, parcel := range parcelContents {
		xy := strings.Split(parcel.ParcelID, ",")
		validateURL := fmt.Sprintf(serverURL+"/validate?x=%s&y=%s", xy[0], xy[1])
		resp, err3 := http.Get(validateURL)
		if err3 != nil {
			log.Fatalf("Failed to get url %s", validateURL)
		}
		defer resp.Body.Close()

		var parcelMetadata handlers.Metadata
		err := json.NewDecoder(resp.Body).Decode(&parcelMetadata)
		if err != nil {
			log.Fatal(err)
		}

		err = client.Set(parcel.ParcelID, parcelMetadata.RootCid, 0).Err()
		if err != nil {
			log.Fatal("Failed to save rootCID to Redis client\n", err)
		}

		err = client.HMSet("metadata_"+parcelMetadata.RootCid, structs.Map(parcelMetadata)).Err()
		if err != nil {
			log.Fatal("Failed to save metadata to Redis client")
		}

		for filePath, cid := range parcel.Contents {
			downloadURL := serverURL + "/contents?" + cid
			store.SaveFile(downloadURL)

			err = client.HSet("content_"+parcelMetadata.RootCid, filePath, cid).Err()
			if err != nil {
				log.Fatal(err)
			}
		}
	}

}

func getServerURL(serverURL string, port string) string {
	serverString := fmt.Sprintf("%s:%s", serverURL, port)
	baseURL, err := url.Parse(serverString)
	if err != nil {
		log.Fatalf("Cannot parse server url: %s", serverString)
	}
	return baseURL.Host
}