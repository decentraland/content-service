package handlers

import (
	"fmt"
	"github.com/decentraland/content-service/storage"
	"github.com/go-redis/redis"
	"github.com/decentraland/content-service/data"
	. "github.com/decentraland/content-service/utils"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"encoding/json"
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
	IsValidParcel(pid string) (bool, error)
}

type MappingsServiceImpl struct {
	RedisClient data.RedisClient
	Dcl         data.Decentraland
	Storage 	storage.Storage
}

func NewMappingsService(client data.RedisClient, dcl data.Decentraland, storage storage.Storage) *MappingsServiceImpl {
	return &MappingsServiceImpl{client, dcl, storage}
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

func (ms *MappingsServiceImpl) IsValidParcel(pid string) (bool, error) {
	val, err := ms.RedisClient.ProcessedParcel(pid)
	if err != nil {
		return false, err
	}
	if val {
		return true, nil
	}
	// All this method should not be maintained, that's a reason to put everything here. It should eventually removed,
	// but not maintened (unless bugs, of course)
	// Here we will find scene.json of a parcel given its position and check if all the parcels that figure in the
	// scene.json still points to that scene. A parcel is valid only if that happens (or if it points ot a newer scene,
	// which should've been checked already)
	info, err := ms.GetParcelInformation(pid)
	if (err != nil && err != redis.Nil) {
		return false, err
	}
	if info == nil || info.Contents == nil {
		return false, fmt.Errorf("Can't find content on parcel info %s", pid)
	}
	log.Info(info)
	scene := ""
	for _, ce := range info.Contents {
		if strings.Contains(ce.File, "scene.json") {
			scene = ce.Cid
			break
		}
	}
	if scene == "" {
		return false, fmt.Errorf("Can't find scene.json on parcel info %s", pid)
	}
	if ms.Storage == nil {
		return false, fmt.Errorf("No storage found on mapping service")
	}
	sceneUrl := ms.Storage.GetFile(scene)

	sceneJson, err := http.Get(sceneUrl)
	if err != nil {
		return false, err
	}

	var sceneMap map[string]interface{}
	bytes, err := ioutil.ReadAll(sceneJson.Body)
	if err != nil {
		return false, fmt.Errorf("Can't read scene.json content for parcel", pid)
	}
	err = json.Unmarshal(bytes, &sceneMap)
	if err != nil {
		return false, fmt.Errorf("Can't parse scene.json for parcel %s", pid)
	}

	sceneValue, ok := sceneMap["scene"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("can't find scene info in scene.json for parcel %s", pid)
	}

	parcels, ok := sceneValue["parcels"].([]string)
	if !ok {
		return false, fmt.Errorf("can't parse parcels in scene.json for parcel %s", pid)
	}

	allValid := true
	for _, p := range parcels {
		parcelCid, err := ms.RedisClient.GetParcelInfo(p)
		if err != nil && err != redis.Nil {
			return false, err
		}
		if parcelCid != info.RootCID {
			allValid = false
			break
		}
	}

	if !allValid {
		_ = ms.RedisClient.SetProcessedParcel(pid)
		return false, nil
	}

	for _, p := range parcels {
		_ = ms.RedisClient.SetProcessedParcel(p)
	}

	return true, nil
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
			// TODO: Get parcel info v2
			// TODO: Get scene.json if needed
			continue
		}
		if err != nil {
			return nil, err //TODO handle??
		}

		validParcel, err := ms.IsValidParcel(pid)
		if err != nil {
			continue //TODO handle??
		}
		if !validParcel {
			continue
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

func (ms *MappingsServiceImpl) GetSceneInformation(cid string) (*ParcelContent, error) {
	parcels, err := ms.RedisClient.GetSceneParcels(cid)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	if len(parcels) == 0 {
		//get parcels somehow
	}

	info, err := ms.GetParcelInformation(parcels[0])
	return info, err
}