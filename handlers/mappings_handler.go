package handlers

import (
	"fmt"
	"github.com/decentraland/content-service/data"
	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
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
	parcels, estates, err := ms.Dcl.GetMap(x1, y1, x2, y2)
	if err != nil {
		return nil, WrapInInternalError(err)
	}
	var mapContents []ParcelContent
	for k := range consolidateParcelsIds(parcels, estates) {
		content, err := ms.GetParcelInformation(k)
		if err != nil {
			return nil, WrapInInternalError(err)
		}
		if content != nil {
			mapContents = append(mapContents, *content)
		}
	}
	return mapContents, nil
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

	metadata, err := ms.RedisClient.GetParcelMetadata(parcelId)
	if metadata == nil || err != nil {
		return nil, err
	}
	return &ParcelContent{ParcelID: parcelId, Contents: content, RootCID: metadata["root_cid"].(string), Publisher: metadata["pubkey"].(string)}, nil
}

func consolidateParcelsIds(parcels []*data.Parcel, estates []*data.Estate) map[string]struct{} {
	parcelsId := make(map[string]struct{})

	appendParcels(parcelsId, &parcels, func(p *data.Parcel) string {
		return p.ID
	})

	onlyCoords := func(p *data.Parcel) string {
		return fmt.Sprintf("%d,%d", p.X, p.Y)
	}

	for _, estate := range estates {
		appendParcels(parcelsId, &estate.Data.Parcels, onlyCoords)
	}
	return parcelsId
}

func appendParcels(result map[string]struct{}, parcels *[]*data.Parcel, idExtractor func(p *data.Parcel) string) {
	for _, p := range *parcels {
		id := idExtractor(p)
		if _, ok := result[id]; !ok {
			result[id] = struct{}{}
		}
	}
}
