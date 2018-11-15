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

func GetMappings(ctx interface{}, r *http.Request) (Response, error) {
	ms, ok := ctx.(MappingsService)
	if !ok {
		return nil, NewInternalError("Invalid Configuration")
	}

	params, err := mapValuesToInt(mux.Vars(r))
	if err != nil {
		return nil, err
	}

	mapContents, err := ms.GetMappings(params["x1"], params["y1"], params["x2"], params["y2"])
	if err != nil {
		return nil, err
	}
	if mapContents == nil {
		return NewOkEmptyResponse(), nil
	}
	return NewOkJsonResponse(mapContents), nil
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

// Logic layer

type MappingsService interface {
	GetMappings(x1, y1, x2, y2 int) ([]ParcelContent, error)
	GetParcelInformation(parcelId string) (ParcelContent, error)
}

type MappingsServiceImpl struct {
	RedisClient data.RedisClient
	Dcl         data.Decentraland
}

func NewMappingsService(client data.RedisClient, dcl data.Decentraland) *MappingsServiceImpl {
	return &MappingsServiceImpl{client, dcl}
}

func (ms *MappingsServiceImpl) GetMappings(x1, y1, x2, y2 int) ([]ParcelContent, error) {
	parcels, estates, err := ms.Dcl.GetMap(x1, y1, x2, y2)
	if err != nil {
		return nil, WrapInInternalError(err)
	}

	for _, estate := range estates {
		parcels = append(parcels, estate.Data.Parcels...)
	}

	var mapContents []ParcelContent
	for _, parcel := range parcels {
		content, err := ms.GetParcelInformation(parcel.ID)
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
func (ms *MappingsServiceImpl) GetParcelInformation(parcelId string) (ParcelContent, error) {
	var pc ParcelContent
	content, err := ms.RedisClient.GetParcelContent(parcelId)

	if err == redis.Nil {
		return pc, nil
	} else if err != nil {
		return pc, err
	}
	metadata, err := ms.RedisClient.GetParcelMetadata(parcelId)
	if err != nil {
		return pc, err
	}
	return ParcelContent{ParcelID: parcelId, Contents: content, RootCID: metadata["root_cid"].(string), Publisher: metadata["pubkey"].(string)}, nil
}
