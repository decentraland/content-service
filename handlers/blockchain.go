package handlers

import (
	"encoding/json"
	"fmt"
	"github.com/decentraland/content-service/config"
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

func getParcel(x, y int, config *config.DecentralandApi) (*parcel, error) {
	u, _ := url.Parse(config.LandUrl)
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

func getEstate(id int, config *config.DecentralandApi) (*estate, error) {
	u, _ := url.Parse(config.LandUrl)
	u.Path = path.Join(u.Path, fmt.Sprintf("estates/%d", id))
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

func getMap(x1, y1, x2, y2 int, config *config.DecentralandApi) ([]*parcel, []*estate, error) {
	u, _ := url.Parse(config.LandUrl)
	u.Path = path.Join(u.Path, fmt.Sprintf("map?nw=%d,%d&se=%d,%d", x1, y1, x2, y2))
	url, _ := url.PathUnescape(u.String())
	resp, err := http.Get(url)
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
