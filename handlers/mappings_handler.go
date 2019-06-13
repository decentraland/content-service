package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/storage"
	. "github.com/decentraland/content-service/utils"
	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
)

type ParcelContent struct {
	ParcelID  string            `json:"parcel_id"`
	Contents  []*ContentElement `json:"contents"`
	RootCID   string            `json:"root_cid"`
	Publisher string            `json:"publisher"`
}

type SceneContent struct {
	RootCID  string         `json:"root_cid"`
	SceneCID string         `json:"scene_cid"`
	Content  *ParcelContent `json:"content"`
}

type Scene struct {
	ParcelId string `json:"parcel_id"`
	RootCID  string `json:"root_cid"`
	SceneCID string `json:"scene_cid"`
}

type ContentElement struct {
	File string `json:"file"`
	Cid  string `json:"hash"`
}

type StringPair struct {
	A string
	B string
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
	GetScenes(x1, y1, x2, y2 int) ([]*Scene, error)
	GetParcelInformation(parcelId string) (*ParcelContent, error)
	GetInfo(cid []string) ([]*SceneContent, error)
}

type MappingsServiceImpl struct {
	RedisClient data.RedisClient
	Dcl         data.Decentraland
	Storage     storage.Storage
}

func NewMappingsService(client data.RedisClient, dcl data.Decentraland, storage storage.Storage) *MappingsServiceImpl {
	return &MappingsServiceImpl{client, dcl, storage}
}

func (ms *MappingsServiceImpl) GetMappings(x1, y1, x2, y2 int) ([]ParcelContent, error) {

	log.Debugf("Retrieving map information within coordinates (%d,%d) and (%d,%d)", x1, y1, x2, y2)
	parcels := RectToParcels(x1, y1, x2, y2, 200)

	if parcels == nil {
		return nil, fmt.Errorf("Too many parcels requested")
	}
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

func (ms *MappingsServiceImpl) GetScenes(x1, y1, x2, y2 int) ([]*Scene, error) {
	log.Debugf("Retrieving map information within points (%d, %d, %d, %d)", x1, x2, y1, y2)

	pids := RectToParcels(x1, y1, x2, y2, 200)
	if pids == nil {
		return nil, fmt.Errorf("too many parcels requested")
	}
	cids := make(map[string]bool, len(pids))

	for _, pid := range pids {
		cid, err := ms.RedisClient.GetParcelCID(pid)
		if cid == "" {
			continue
		}
		if err != nil {
			return nil, err
		}

		validParcel, err := ms.RedisClient.ProcessedParcel(pid)
		if err != nil {
			log.Errorf("error when checking validity of parcel %s", pid)
			// skip on error
		}
		if !validParcel {
			continue
		}

		cids[cid] = true
	}

	ret := make([]*Scene, 0, len(cids))
	for cid, _ := range cids {
		parcels, err := ms.RedisClient.GetSceneParcels(cid)

		if err != nil && err != redis.Nil {
			return nil, fmt.Errorf("can't read parcels for a scene because: %s", err)
		}
		sceneCID, err := ms.RedisClient.GetSceneCid(cid)
		if err != nil && err != redis.Nil {
			log.Errorf("error reading scene cid for cid %s", cid)
			// we just use the empty string in this case
		}

		for _, p := range parcels {
			ret = append(ret, &Scene{
				SceneCID: sceneCID,
				RootCID:  cid,
				ParcelId: p,
			})
		}
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

func (s *MappingsServiceImpl) GetInfo(cids []string) ([]*SceneContent, error) {
	parcels := make(map[string]*StringPair, len(cids))
	for _, cid := range cids {
		ps, err := s.RedisClient.GetSceneParcels(cid)
		if err != nil && err != redis.Nil {
			return nil, err
		}
		sceneCID := ""
		rootCID := ""
		if ps == nil || len(ps) == 0 {
			// Maybe the parameter is not the root cid, but the scene cid, which we will be eventually support better
			rootCID, err = s.RedisClient.GetRootCid(cid)
			if err != nil && err != redis.Nil {
				log.Errorf("error when getting rootcid for hash %s with error %s", cid, err)
				continue
			}
			ps, err = s.RedisClient.GetSceneParcels(rootCID)
			if err != nil && err != redis.Nil {
				log.Errorf("error when reading parcels for cid %s with error %s", rootCID, err)
				continue
			}
			if ps == nil || len(ps) == 0 {
				continue
			}
			sceneCID = cid
		}

		if sceneCID == "" {
			sceneCID, _ = s.RedisClient.GetSceneCid(cid)
			rootCID = cid
		}

		parcels[rootCID] = &StringPair{A: ps[0], B: sceneCID}
	}

	ret := make([]*SceneContent, 0, len(cids))
	for k, v := range parcels {
		content, err := s.GetParcelInformation(v.A)
		if err != nil {
			log.Errorf("error getting information for parcel %s with error %s", v.A, err)
			continue
		}

		ret = append(ret, &SceneContent{
			RootCID:  k,
			SceneCID: v.B,
			Content:  content,
		})
	}

	return ret, nil
}

func GetInfo(ctx interface{}, r *http.Request) (Response, error) {
	ms, ok := ctx.(MappingsService)
	if !ok {
		log.Fatal("Invalid Handler configuration")
		return nil, NewInternalError("Invalid Configuration")
	}

	params := mux.Vars(r)

	cidsParam := params["cids"]
	cids := strings.Split(cidsParam, ",")

	ret, err := ms.GetInfo(cids)
	if err != nil {
		return nil, err
	}
	return NewOkJsonResponse(ret), nil
}
