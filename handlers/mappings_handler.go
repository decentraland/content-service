package handlers

import (
	"github.com/go-redis/redis"
	"github.com/decentraland/content-service/data"
	. "github.com/decentraland/content-service/utils"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

type ParcelContent struct {
	ParcelID  string            `json:"parcel_id"`
	Contents  []*ContentElement `json:"contents"`
	RootCID   string            `json:"root_cid"`
	Publisher string            `json:"publisher"`
}

type ContentElement struct {
	File string `json:"file"`
	Cid  string `json:"hash"`
}

func GetMappings(ctx interface{}, r *http.Request) (Response, error) {
	ms, ok := ctx.(MappingsService)
	if !ok {
		log.Fatal("Invalid Handler configuration")
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

func GetScenes(ctx interface{}, r *http.Request) (Response, error) {
	ms, ok := ctx.(MappingsService)
	if !ok {
		log.Fatal("Invalid Handler configuration")
		return nil, NewInternalError("Invalid Configuration")
	}

	params, err := mapValuesToInt(mux.Vars(r))
	if err != nil {
		return nil, err
	}

	mapContents, err := ms.GetScenes(params["x1"], params["y1"], params["x2"], params["y2"])
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
	GetScenes(x1, y1, x2, y2 int) ([]map[string]string, error)
	GetParcelInformation(parcelId string) (*ParcelContent, error)
}

type MappingsServiceImpl struct {
	RedisClient data.RedisClient
	Dcl         data.Decentraland
}

func NewMappingsService(client data.RedisClient, dcl data.Decentraland) *MappingsServiceImpl {
	return &MappingsServiceImpl{client, dcl}
}

func (ms *MappingsServiceImpl) GetMappings(x1, y1, x2, y2 int) ([]ParcelContent, error) {

	log.Debugf("Retrieving map information within coordinates (%d,%d) and (%d,%d)", x1, y1, x2, y2)
	parcels := RectToParcels(x1, y1, x2, y2)

	mapContents := []ParcelContent{}
	for _, pid := range parcels {
		content, err := ms.GetParcelInformation(pid)
		if err != nil {
			return nil, WrapInInternalError(err)
		}
		if content != nil {
			mapContents = append(mapContents, *content)
		}
	}
	return mapContents, nil
}

func (ms *MappingsServiceImpl) GetScenes(x1, y1, x2, y2 int) ([]map[string]string, error ) {
	log.Debugf("Retrieving map information within points (%d, %d, %d, %d)", x1, x2, y1, y2)

	// we will need to move this down later
	parcelMap := make(map[string]string, 0)

	pids := RectToParcels(x1, y1, x2, y2)
	cids := make(map[string]bool, len(pids))
	for _, pid := range pids {
		cid, err := ms.RedisClient.GetParcelInfo(pid)
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, err //TODO handle??
		}
		parcelMap[pid] = cid //TODO: This is for compatibility with old queries, should be removed _eventually_
		cids[cid] = true
	}


	for cid, _ := range cids {
		parcels, err := ms.RedisClient.GetSceneParcels(cid)
		if err != nil && err != redis.Nil {
			return nil, err //TODO handle??
		}
		for _, p := range parcels {
			parcelMap[p] = cid
		}
	}

	// Ugly and inefficient, but client requires an array. This can be improved once we remove the TODO above and we are sure that elements are not repeated
	ret := make([]map[string]string, 0, len(parcelMap))
	for k, v := range parcelMap {
		m := make(map[string]string, 1)
		m[k] = v
		ret = append(ret, m)
	}
	return ret, nil
}

/**
Retrieves the consolidated information of a given Parcel <ParcelContent>
if the parcel does not exists, the ParcelContent.Contents will be nil
*/
func (ms *MappingsServiceImpl) GetParcelInformation(parcelId string) (*ParcelContent, error) {
	content, err := ms.RedisClient.GetParcelContent(parcelId)
	if content == nil || err != nil {
		return nil, err
	}

	var elements []*ContentElement

	for name, cid := range content {
		elements = append(elements, &ContentElement{File: name, Cid: cid})
	}

	metadata, err := ms.RedisClient.GetParcelMetadata(parcelId)
	if metadata == nil || err != nil {
		return nil, err
	}
	return &ParcelContent{ParcelID: parcelId, Contents: elements, RootCID: metadata["root_cid"].(string), Publisher: metadata["pubkey"].(string)}, nil
}
