package data

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/decentraland/content-service/metrics"
	"github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"
)

type parcelResponse struct {
	Ok   bool    `json:"ok"`
	Data *Parcel `json:"data"`
}

type estateResponse struct {
	Ok   bool    `json:"ok"`
	Data *Estate `json:"data"`
}

type accessResponse struct {
	Ok   bool        `json:"ok"`
	Data *AccessData `json:"data"`
}

type AccessData struct {
	Id               string
	Address          string
	IsApprovedForAll bool
	IsOwner          bool
	IsOperator       bool
	IsUpdateOperator bool
}

func (ad *AccessData) CheckAccess() bool {
	return ad.IsApprovedForAll || ad.IsOperator || ad.IsOwner || ad.IsUpdateOperator
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
	GetParcelAccessData(address string, x int64, y int64) (*AccessData, error)
}

type DclClient struct {
	ApiUrl string
	Agent  *metrics.Agent
}

func NewDclClient(apiUrl string, agent *metrics.Agent) *DclClient {
	return &DclClient{apiUrl, agent}
}

// Retrieves a accessData information from Decentraland
func (dcl DclClient) GetParcel(x, y int) (*Parcel, error) {
	var jsonResponse parcelResponse
	err := dcl.doGet(buildUrl(dcl.ApiUrl, "parcels/%d/%d", x, y), &jsonResponse)
	if err != nil {
		return nil, err
	}

	return jsonResponse.Data, nil
}

//Retrieves the Estate by its Id
func (dcl DclClient) GetEstate(id int) (*Estate, error) {
	var jsonResponse estateResponse
	err := dcl.doGet(buildUrl(dcl.ApiUrl, "estates/%d", id), &jsonResponse)
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
	err := dcl.doGet(buildUrl(dcl.ApiUrl, "map?nw=%d,%d&se=%d,%d", x1, y1, x2, y2), &jsonResponse)
	if err != nil {
		return nil, nil, err
	}

	return jsonResponse.Data.Assets.Parcels, jsonResponse.Data.Assets.Estates, nil
}

// Retrieves the access data of a address over a given accessData
func (dcl DclClient) GetParcelAccessData(address string, x int64, y int64) (*AccessData, error) {
	var response accessResponse
	err := dcl.doGet(buildUrl(dcl.ApiUrl, "parcels/%d/%d/%s/authorizations", x, y, address), &response)
	if err != nil {
		return nil, err
	}

	return response.Data, nil
}

func buildUrl(basePath string, relPath string, args ...interface{}) string {
	u, _ := url.Parse(basePath)
	u.Path = path.Join(u.Path, fmt.Sprintf(relPath, args...))
	urlResult, _ := url.PathUnescape(u.String())
	return urlResult
}

func (dcl DclClient) doGet(url string, response interface{}) error {
	t := time.Now()
	resp, err := http.Get(url)
	dcl.Agent.RecordDCLResponseTime(time.Since(t))
	if err != nil {
		logrus.Errorf("Failed to retrieve information from URL[%s]: %s", url, err.Error())
		return err
	}
	if resp.StatusCode >= 400 {
		logrus.Errorf("[DCL API FAILED] Request failed to URL[%s] with Status[%d]: %s", url, resp.StatusCode, bodyToString(resp))
		dcl.Agent.RecordDCLAPIError(resp.StatusCode)
		return errors.New("DCL Replied with an error")
	}
	return json.NewDecoder(resp.Body).Decode(response)
}

func bodyToString(r *http.Response) string {
	defer r.Body.Close()
	respBytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return ""
	}
	return string(respBytes)
}
