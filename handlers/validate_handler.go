package handlers

import (
	"fmt"
	"github.com/decentraland/content-service/data"
	"net/http"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
)

type ValidateParcelCtx struct {
	RedisClient data.RedisClient
}

func GetParcelMetadata(ctx interface{}, r *http.Request) (Response, error) {
	c, ok := ctx.(ValidateParcelCtx)
	if !ok {
		return nil, NewInternalError("Invalid Configuration")
	}

	params := mux.Vars(r)

	parcelMeta, err := getParcelMetadata(c.RedisClient, fmt.Sprintf("%+s,%+s", params["x"], params["y"]))
	if err != nil {
		return nil, err
	}

	if parcelMeta == nil {
		return NewOkEmptyResponse(), nil
	}
	return NewOkJsonResponse(parcelMeta), nil
}

// Retrieves the Parcel metadata or an error if no metadata for the given id is found
func getParcelMetadata(rc data.RedisClient, parcelId string) (map[string]interface{}, error) {
	parcelMeta, err := rc.GetParcelMetadata(parcelId)
	if err == redis.Nil {
		return nil, NewNotFoundError("Parcel metadata not found")
	} else if err != nil {
		return nil, WrapInInternalError(err)
	}
	return parcelMeta, nil
}
