package handlers

import (
	"fmt"
	"github.com/decentraland/content-service/data"
	"net/http"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

func GetParcelMetadata(ctx interface{}, r *http.Request) (Response, error) {
	ms, ok := ctx.(MetadataService)
	if !ok {
		return nil, NewInternalError("Invalid Configuration")
	}

	params := mux.Vars(r)

	parcelMeta, err := ms.GetParcelMetadata(fmt.Sprintf("%+s,%+s", params["x"], params["y"]))
	if err != nil {
		return nil, WrapInInternalError(err)
	}

	if parcelMeta == nil {
		return nil, NewNotFoundError("Parcel metadata not found")
	}

	return NewOkJsonResponse(parcelMeta), nil
}

// Logic Layer

type MetadataService interface {
	GetParcelMetadata(parcelId string) (map[string]interface{}, error)
}

type MetadataServiceImpl struct {
	RedisClient data.RedisClient
}

func NewMetadataService(client data.RedisClient) *MetadataServiceImpl {
	return &MetadataServiceImpl{client}
}

// Retrieves the Parcel metadata for the given id is found
func (ms *MetadataServiceImpl) GetParcelMetadata(parcelId string) (map[string]interface{}, error) {
	parcelMeta, err := ms.RedisClient.GetParcelMetadata(parcelId)
	if err == redis.Nil {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return parcelMeta, nil
}
