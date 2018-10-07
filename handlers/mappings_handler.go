package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

type MappingsHandler struct {
	RedisClient *redis.Client
}

func (handler *MappingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	paramsInt, err := mapValuesToInt(params)

	parcels, estates, err := getMap(paramsInt["x1"], paramsInt["y1"], paramsInt["x2"], paramsInt["y2"])
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	for _, estate := range estates {
		parcels = append(parcels, estate.Data.Parcels...)
	}

	parcelsContent := make(map[string]map[string]string)
	for _, parcel := range parcels {
		parcelContent, err := getParcelContent(handler.RedisClient, parcel.ID)
		// If parcel is not found ignore and keep going
		if err == redis.Nil {
			continue
		} else if err != nil {
			log.Println(err)
			http.Error(w, http.StatusText(500), 500)
			return
		}

		parcelsContent[parcel.ID] = parcelContent
	}

	contentsJSON, err := json.Marshal(parcelsContent)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	_, err = w.Write(contentsJSON)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
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
