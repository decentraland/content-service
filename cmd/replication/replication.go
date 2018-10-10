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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/fatih/structs"
	"github.com/go-redis/redis"
	"github.com/spf13/viper"
)

type configuration struct {
	Server struct {
		URL  string
		Port string
	}
	S3Storage struct {
		Bucket string
		ACL string
	}
	LocalStorage string

	Redis struct {
		Address  string
		Password string
		DB       int
	}
}

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
	RootCid      string `json:"rootcid" structs:"rootcid"`
}

var config configuration
var client *redis.Client

func init() {
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	err := viper.Unmarshal(&config)
	if err != nil {
		log.Fatalf("Unable to parse config file, %s", err)
	}
	
	client = redis.NewClient(&redis.Options{
		Addr:     config.Redis.Address,
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})
}

func main() {	
	serverURL := getServerURL(config.Server.URL, config.Server.Port)
	mappingsURL := serverURL + "/mappings?nw=-150,150&se=150,-150"
	resp, err := http.Get(mappingsURL)
	if err != nil {
		log.Fatalf("Failed to get url %s", mappingsURL)
	}
	defer resp.Body.Close()

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

		var parcelMetadata metadata
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
			if config.S3Storage.Bucket != "" {
				err := saveFileS3(filePath, downloadURL)
				if err != nil {
					log.Fatal(err)
				}
			} else {
				localPath := filepath.Join(config.LocalStorage, parcelMetadata.RootCid, filePath)
				err := saveFileLocal(localPath, downloadURL)
				if err != nil && err != io.EOF {
					log.Fatalf("Cannot save file %s to local storage", localPath)
				}
			}

			err = client.HSet("content_"+parcelMetadata.RootCid, filePath, cid).Err()
			if err != nil {
				log.Fatal(err)
			}
		}
	}

}

func getServerURL(serverURL string, port string) string {
	baseURL, err := url.Parse(serverURL)
	if err != nil {
		log.Fatalf("Cannot parse server url: %s", serverURL)
	}
	if baseURL.Scheme == "" {
		baseURL.Scheme = "http"
	}
	urlString := baseURL.String()
	if port != "" {
		urlString = fmt.Sprintf("%s:%s", urlString, port)
	}
	return urlString
}

func saveFileS3(filePath string, downloadURL string) error {
	resp, err := http.Get(downloadURL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	sess := session.Must(session.NewSession())

	uploader := s3manager.NewUploader(sess)

	_, err2 := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(config.S3Storage.Bucket),
		Key:    aws.String(filePath),
		ACL:    aws.String(config.S3Storage.Bucket),
		Body:   resp.Body,
	})

	return err2
}

func saveFileLocal(localPath string, downloadURL string) error {
	err := os.MkdirAll(filepath.Dir(localPath), os.ModePerm)
	if err != nil {
		return err
	}

	file, err2 := os.Create(localPath)
	if err2 != nil {
		return err2
	}
	defer file.Close()

	resp, err3 := http.Get(downloadURL)
	if err3 != nil {
		return err3
	}
	defer resp.Body.Close()

	_, err4 := io.Copy(file, resp.Body)
	return err4
}