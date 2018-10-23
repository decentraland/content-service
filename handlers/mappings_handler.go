package handlers

import (
	"encoding/json"
	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"net/http"
	"strconv"
)

type ParcelContent struct {
	ParcelID  string            `json:"parcel_id"`
	Contents  map[string]string `json:"contents"`
	RootCID   string            `json:"root_cid"`
	Publisher string            `json:"publisher"`
}

type MappingsHandler struct {
	RedisClient *redis.Client
}

func (handler *MappingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	paramsInt, err := mapValuesToInt(params)

	parcels, estates, err := getMap(paramsInt["x1"], paramsInt["y1"], paramsInt["x2"], paramsInt["y2"])
	if err != nil {
		handle500(w, err)
		return
	}

	for _, estate := range estates {
		parcels = append(parcels, estate.Data.Parcels...)
	}

	var mapContents []ParcelContent
	for _, parcel := range parcels {
		content, err := getParcelInformation(handler.RedisClient, parcel.ID)
		if err != nil {
			handle500(w, err)
			return
		}
		if content.Contents != nil {
			mapContents = append(mapContents, content)
		}
	}

	contentsJSON, err := json.Marshal(mapContents)
	if err != nil {
		handle500(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	_, err = w.Write(contentsJSON)
	if err != nil {
		handle500(w, err)
		return
	}
}

/**
Retrieves the consolidated information of a given Parcel <ParcelContent>
if the parcel does not exists, the ParcelContent.Contents will be nil
*/
func getParcelInformation(client *redis.Client, parcelId string) (ParcelContent, error) {
	var pc ParcelContent
	content, err := getParcelContent(client, parcelId)

	if err == redis.Nil {
		return pc, nil
	} else if err != nil {
		return pc, err
	}
	metadata, err := getParcelMetadata(client, parcelId)
	if err != nil {
		return pc, err
	}
	return ParcelContent{ParcelID: parcelId, Contents: content, RootCID: metadata["root_cid"].(string), Publisher: metadata["pubkey"].(string)}, nil
}

func mapValuesToInt(mapStr map[string]string) (map[string]int, error) {
	// var mapInt map[string]int
	var err error
	mapInt := make(map[string]int)
	for k, v := range mapStr {
		mapInt[k], err = strconv.Atoi(v)
		if err != nil {
			return nil, err
		}
	}

	return mapInt, nil
}
