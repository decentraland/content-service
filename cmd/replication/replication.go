package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/decentraland/content-service/metrics"

	"github.com/decentraland/content-service/config"
	"github.com/decentraland/content-service/internal/handlers"
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
	var x1, y1, x2, y2, url string
	args := os.Args[1:]
	if len(args) == 5 {
		x1, y1, x2, y2, url = args[0], args[1], args[2], args[3], args[4]
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

	agent, _ := metrics.Make(config.Metrics{AnalyticsKey: "", Enabled: false, AppName: ""})
	sto := storage.NewStorage(&conf.Storage, agent)

	mappingsURL := fmt.Sprintf("%smappings?nw=%s,%s&se=%s,%s", url, x1, y1, x2, y2)
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
		validateURL := fmt.Sprintf("%svalidate?x=%s&y=%s", url, xy[0], xy[1])
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

		for _, element := range parcel.Contents {
			downloadURL := fmt.Sprintf("%scontents?%s", url, element.Cid)
			resp, err := http.Get(downloadURL)
			if err != nil {
				log.Fatal(err)
			}
			defer resp.Body.Close()

			_, err = sto.SaveFile(element.Cid, resp.Body, resp.Header.Get("Content-Type"))
			if err != nil {
				log.Fatal(err)
			}

			err = client.HSet("content_"+parcelMetadata.RootCid, element.File, element.Cid).Err()
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}
