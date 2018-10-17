package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"os"

	"github.com/decentraland/content-service/config"
	"github.com/decentraland/content-service/handlers"
	"github.com/decentraland/content-service/storage"
	"github.com/fatih/structs"
	"github.com/go-redis/redis"
)

var conf *config.Configuration
var client *redis.Client

func init() {
	conf = config.GetConfig("config")
	
	client = redis.NewClient(&redis.Options{
		Addr:     conf.Redis.Address,
		Password: conf.Redis.Password,
		DB:       conf.Redis.DB,
	})
}

func main() {
	var x1, y1, x2, y2 string
	args := os.Args[1:]
	if len(args) == 4 {
		x1, y1, x2, y2 = args[0], args[1], args[2], args[3]
	} else {
		fmt.Println("Input NW coordinates")
		fmt.Print("x1 = ")
		fmt.Scanln(&x1)
		fmt.Print("y1 = ")
		fmt.Scanln(&y1)
		fmt.Println("Input SE coordinates")
		fmt.Print("x2 = ")
		fmt.Scanln(&x2)
		fmt.Print("y2 = ")
		fmt.Scanln(&y2)
	}

	sto := storage.NewStorage(conf)

	serverURL := config.GetServerAddress(conf.Server.Hostname, conf.Server.Port)
	mappingsURL := fmt.Sprintf("http://%s/mappings?nw=%s,%s&se=%s,%s", serverURL, x1, y1, x2, y2)
	resp, err := http.Get(mappingsURL)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	var parcelContents []handlers.ParcelContent
	err = json.NewDecoder(resp.Body).Decode(&parcelContents)
	if err != nil {
		log.Fatal(err)
	}

	for _, parcel := range parcelContents {
		xy := strings.Split(parcel.ParcelID, ",")
		validateURL := fmt.Sprintf("http://%s/validate?x=%s&y=%s", serverURL, xy[0], xy[1])
		resp, err3 := http.Get(validateURL)
		if err3 != nil {
			log.Fatal(err)
		}
		defer resp.Body.Close()

		var parcelMetadata handlers.Metadata
		err := json.NewDecoder(resp.Body).Decode(&parcelMetadata)
		if err != nil {
			log.Fatal(err)
		}

		err = client.Set(parcel.ParcelID, parcelMetadata.RootCid, 0).Err()
		if err != nil {
			log.Fatal(err)
		}

		err = client.HMSet("metadata_"+parcelMetadata.RootCid, structs.Map(parcelMetadata)).Err()
		if err != nil {
			log.Fatal(err)
		}

		for filePath, cid := range parcel.Contents {
			downloadURL := fmt.Sprintf("http://%s/contents?%s", serverURL, cid)
			resp, err := http.Get(downloadURL)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			_, err = sto.SaveFile(filePath, resp.Body)
			if err != nil {
				log.Fatal(err)
			}

			err = client.HSet("content_"+parcelMetadata.RootCid, filePath, cid).Err()
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}