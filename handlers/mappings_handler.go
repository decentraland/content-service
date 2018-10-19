package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

type ParcelContent struct {
	ParcelID string            `json:"parcel_id"`
	Contents map[string]string `json:"contents"`
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
		contents, err := getParcelContent(handler.RedisClient, parcel.ID)
		// If parcel is not found ignore and keep going
		if err == redis.Nil {
			continue
		} else if err != nil {
			handle500(w, err)
			return
		}

		mapContents = append(mapContents, ParcelContent{ParcelID: parcel.ID, Contents: contents})
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
