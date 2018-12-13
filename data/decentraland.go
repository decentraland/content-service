package data

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"path"
)

type parcelResponse struct {
	Ok   bool    `json:"ok"`
	Data *Parcel `json:"data"`
}

type estateResponse struct {
	Ok   bool    `json:"ok"`
	Data *Estate `json:"data"`
}

type MapResponse struct {
	Ok   bool `json:"ok"`
	Data struct {
		Assets struct {
			Parcels []*Parcel `json:"parcels"`
			Estates []*Estate `json:"estates"`
		} `json:"assets"`
	} `json:"data"`
}

type Parcel struct {
	ID             string `json:"id"`
	X              int    `json:"x"`
	Y              int    `json:"y"`
	Owner          string `json:"owner"`
	UpdateOperator string `json:"update_operator"`
	EstateID       string `json:"estate_id"`
}

type Estate struct {
	ID             string `json:"id"`
	Owner          string `json:"owner"`
	UpdateOperator string `json:"update_operator"`
	Data           struct {
		Parcels []*Parcel `json:"parcels"`
	} `json:"data"`
}

type Decentraland interface {
	GetParcel(x, y int) (*Parcel, error)
	GetEstate(id int) (*Estate, error)
	GetMap(x1, y1, x2, y2 int) ([]*Parcel, []*Estate, error)
}

type DclClient struct {
	ApiUrl string
}

func NewDclClient(apiUrl string) *DclClient {
	return &DclClient{apiUrl}
}

// Retrieves a parcel information from Decentraland
func (dcl DclClient) GetParcel(x, y int) (*Parcel, error) {
	var jsonResponse parcelResponse
	err := doGet(buildUrl(dcl.ApiUrl, "parcels/%d/%d", x, y), &jsonResponse)
	if err != nil {
		return nil, err
	}

	return jsonResponse.Data, nil
}

//Retrieves the Estate by its Id
func (dcl DclClient) GetEstate(id int) (*Estate, error) {
	var jsonResponse estateResponse
	err := doGet(buildUrl(dcl.ApiUrl, "estates/%d", id), &jsonResponse)
	if err != nil {
		return nil, err
	}

	for _, parcel := range jsonResponse.Data.Data.Parcels {
		parcel.ID = fmt.Sprintf("%d,%d", parcel.X, parcel.Y)
	}

	return jsonResponse.Data, nil
}

// Retrieves all parcels information in the given quadrant
func (dcl DclClient) GetMap(x1, y1, x2, y2 int) ([]*Parcel, []*Estate, error) {
	var jsonResponse MapResponse
	err := doGet(buildUrl(dcl.ApiUrl, "map?nw=%d,%d&se=%d,%d", x1, y1, x2, y2), &jsonResponse)
	if err != nil {
		return nil, nil, err
	}

	return jsonResponse.Data.Assets.Parcels, jsonResponse.Data.Assets.Estates, nil
}

func buildUrl(basePath string, relPath string, args ...interface{}) string {
	u, _ := url.Parse(basePath)
	u.Path = path.Join(u.Path, fmt.Sprintf(relPath, args...))
	url, _ := url.PathUnescape(u.String())
	return url
}

func doGet(url string, response interface{}) error {
	resp, err := http.Get(url)
	if err != nil {
		logrus.Errorf("Failed to retrieve information from URL[%s]: %s", url, err.Error())
		return err
	}
	return json.NewDecoder(resp.Body).Decode(response)
}
