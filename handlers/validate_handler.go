package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

type ValidateHandler struct {
	RedisClient *redis.Client
}

func (handler *ValidateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	parcelID := fmt.Sprintf("%+v,%+v", params["x"], params["y"])

	parcelMeta, err := getParcelMetadata(handler.RedisClient, parcelID)
	if err == redis.Nil {
		http.Error(w, http.StatusText(404), 404)
		return
	} else if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	metadataJSON, err := json.Marshal(parcelMeta)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	_, err = w.Write(metadataJSON)
	if err != nil {
		log.Println(err)
		http.Error(w, http.StatusText(500), 500)
		return
	}
}
