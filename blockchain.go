package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type apiResponse struct {
	Data *parcel `json:"data"`
}

type parcel struct {
	ID    string `json:"id"`
	X     int    `json:"x"`
	Y     int    `json:"y"`
	Owner string `json:"owner"`
}

func getParcel(x, y int) (*parcel, error) {
	url := fmt.Sprintf("https://api.decentraland.org/v1/parcels/%d/%d", x, y)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	var data apiResponse
	json.NewDecoder(resp.Body).Decode(&data)
	return data.Data, nil
}
