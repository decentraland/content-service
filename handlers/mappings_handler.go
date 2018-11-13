package handlers

import (
	"github.com/decentraland/content-service/data"
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

type GetMappingsCtx struct {
	RedisClient data.RedisClient
	Dcl         data.Decentraland
}

func GetMappings(ctx interface{}, r *http.Request) (interface{}, error) {
	c, ok := ctx.(GetMappingsCtx)
	if !ok {
		return nil, NewInternalError("Invalid Configuration")
	}

	params := mux.Vars(r)

	paramsInt, err := mapValuesToInt(params)
	if err != nil {
		return nil, err
	}

	mapContents, err := getMappings(c, paramsInt["x1"], paramsInt["y1"], paramsInt["x2"], paramsInt["y2"])
	if err != nil {
		return nil, err
	}
	return mapContents, nil
}

func getMappings(c GetMappingsCtx, x1, y1, x2, y2 int) ([]ParcelContent, error) {
	parcels, estates, err := c.Dcl.GetMap(x1, y1, x2, y2)
	if err != nil {
		return nil, WrapInInternalError(err)
	}

	for _, estate := range estates {
		parcels = append(parcels, estate.Data.Parcels...)
	}

	var mapContents []ParcelContent
	for _, parcel := range parcels {
		content, err := getParcelInformation(c.RedisClient, parcel.ID)
		if err != nil {
			return nil, WrapInInternalError(err)
		}
		if content.Contents != nil {
			mapContents = append(mapContents, content)
		}
	}
	return mapContents, nil
}

/**
Retrieves the consolidated information of a given Parcel <ParcelContent>
if the parcel does not exists, the ParcelContent.Contents will be nil
*/
func getParcelInformation(client data.RedisClient, parcelId string) (ParcelContent, error) {
	var pc ParcelContent
	content, err := client.GetParcelContent(parcelId)

	if err == redis.Nil {
		return pc, nil
	} else if err != nil {
		return pc, err
	}
	metadata, err := client.GetParcelMetadata(parcelId)
	if err != nil {
		return pc, err
	}
	return ParcelContent{ParcelID: parcelId, Contents: content, RootCID: metadata["root_cid"].(string), Publisher: metadata["pubkey"].(string)}, nil
}

func mapValuesToInt(mapStr map[string]string) (map[string]int, error) {
	var err error
	mapInt := make(map[string]int)
	for k, v := range mapStr {
		mapInt[k], err = strconv.Atoi(v)
		if err != nil {
			return nil, WrapInBadRequestError(err)
		}
	}
	return mapInt, nil
}
