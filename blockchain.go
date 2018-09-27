package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type parcelResponse struct {
	Ok   bool    `json:"ok"`
	Data *parcel `json:"data"`
}

type estateResponse struct {
	Ok   bool    `json:"ok"`
	Data *estate `json:"data"`
}

type mapResponse struct {
	Ok   bool `json:"ok"`
	Data struct {
		Assets struct {
			Parcels []*parcel `json:"parcels"`
			Estates []*estate `json:"estates"`
		} `json:"assets"`
	} `json:"data"`
}

type parcel struct {
	ID    string `json:"id"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Owner string `json:"owner"`
}

type estate struct {
	ID    string `json:"id"`
	Owner string `json:"owner"`
}

func getParcel(x, y int) (*parcel, error) {
	url := fmt.Sprintf("https://api.decentraland.org/v1/parcels/%d/%d", x, y)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	var jsonResponse parcelResponse
	json.NewDecoder(resp.Body).Decode(&jsonResponse)
	return jsonResponse.Data, nil
}

func getEstate(id string) (*estate, error) {
	url := fmt.Sprintf("https://api.decentraland.org/v1/estate/%s", id)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	var jsonResponse estateResponse
	json.NewDecoder(resp.Body).Decode(&jsonResponse)
	return jsonResponse.Data, nil
}

func getMap(x1, y1, x2, y2 int) ([]*parcel, []*estate, error) {
	url := fmt.Sprintf("https://api.decentraland.org/v1/map?nw=%d,%d&se=%d,%d", x1, y1, x2, y2)
	resp, err := http.Get(url)
	if err != nil {
		return nil, nil, err
	}

	var jsonResponse mapResponse
	json.NewDecoder(resp.Body).Decode(&jsonResponse)
	return jsonResponse.Data.Assets.Parcels, jsonResponse.Data.Assets.Estates, nil
}
