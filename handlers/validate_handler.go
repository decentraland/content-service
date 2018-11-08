package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/decentraland/content-service/data"
	"net/http"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

type ValidateHandler struct {
	RedisClient data.RedisClient
}

func (handler *ValidateHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	parcelID := fmt.Sprintf("%+s,%+s", params["x"], params["y"])

	parcelMeta, err := handler.RedisClient.GetParcelMetadata(parcelID)
	if err == redis.Nil {
		handle400(w, 404, "Parcel metadata not found")
		return
	} else if err != nil {
		handle500(w, err)
		return
	}

	metadataJSON, err := json.Marshal(parcelMeta)
	if err != nil {
		handle500(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(200)
	_, err = w.Write(metadataJSON)
	if err != nil {
		handle500(w, err)
		return
	}
}
