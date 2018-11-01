package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
)

type parcelResponse struct {
	Ok   bool    `json:"ok"`
	Data *parcel `json:"data"`
}

type estateResponse struct {
	Ok   bool    `json:"ok"`
	Data *estate `json:"data"`
}

type MapResponse struct {
	Ok   bool `json:"ok"`
	Data struct {
		Assets struct {
			Parcels []*parcel `json:"parcels"`
			Estates []*estate `json:"estates"`
		} `json:"assets"`
	} `json:"data"`
}

type parcel struct {
	ID             string `json:"id"`
	X              int    `json:"x"`
	Y              int    `json:"y"`
	Owner          string `json:"owner"`
	UpdateOperator string `json:"update_operator"`
	EstateID       string `json:"estate_id"`
}

type estate struct {
	ID             string `json:"id"`
	Owner          string `json:"owner"`
	UpdateOperator string `json:"update_operator"`
	Data           struct {
		Parcels []*parcel `json:"parcels"`
	} `json:"data"`
}

var landApi string

func getParcel(x, y int) (*parcel, error) {
	u, _ := url.Parse(landApi)
	u.Path = path.Join(u.Path, fmt.Sprintf("parcels/%d/%d", x, y))
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	var jsonResponse parcelResponse
	err = json.NewDecoder(resp.Body).Decode(&jsonResponse)
	if err != nil {
		return nil, err
	}

	return jsonResponse.Data, nil
}

func getEstate(id int) (*estate, error) {
	u, _ := url.Parse(landApi)
	u.Path = path.Join(landApi, fmt.Sprintf("estates/%d", id))
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	var jsonResponse estateResponse
	err = json.NewDecoder(resp.Body).Decode(&jsonResponse)
	if err != nil {
		return nil, err
	}

	for _, parcel := range jsonResponse.Data.Data.Parcels {
		parcel.ID = fmt.Sprintf("%d,%d", parcel.X, parcel.Y)
	}

	return jsonResponse.Data, nil
}

func getMap(x1, y1, x2, y2 int) ([]*parcel, []*estate, error) {
	u, _ := url.Parse(landApi)
	u.Path = path.Join(landApi, fmt.Sprintf("map?nw=%d,%d&se=%d,%d", x1, y1, x2, y2))
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, nil, err
	}

	var jsonResponse MapResponse
	err = json.NewDecoder(resp.Body).Decode(&jsonResponse)
	if err != nil {
		return nil, nil, err
	}

	return jsonResponse.Data.Assets.Parcels, jsonResponse.Data.Assets.Estates, nil
}

// Setup the decentraland server address.
func InitDclClient(address string) {
	landApi = address
}
