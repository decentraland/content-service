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
