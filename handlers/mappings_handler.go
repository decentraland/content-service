package handlers

import (
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

type Scenes struct {
	ParcelID string `json:"parcel_id"`
	RootCID string `json:"root_cid"`
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
	GetScenes(x1, y1, x2, y2 int) ([]Scenes, error)
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

func (ms *MappingsServiceImpl) GetScenes(x1, y1, x2, y2 int) ([]Scenes, error ) {
	log.Debugf("Retrieving map information within points (%d, %d, %d, %d)", x1, x2, y1, y2)

	pids := RectToParcels(x1, y1, x2, y2)
	cids := make(map[string]bool, len(pids))
	for _, pid := range pids {
		cid, err := ms.RedisClient.GetParcelInfo(pid)
		if err != nil {
			return nil, err //TODO handle??
		}
		cids[cid] = true
	}

	parcelMap := make([]Scenes, 0, len(pids))
	for cid, _ := range cids {
		parcels, err := ms.RedisClient.GetSceneParcels(cid)
		if err != nil {
			return nil, err //TODO handle??
		}
		for _, p := range parcels {
			parcelMap = append(parcelMap, Scenes{ParcelID: p, RootCID:cid})
		}
	}

	return parcelMap, nil
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
