package decentraland

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"path"
	"time"

	"github.com/decentraland/content-service/internal/metrics"
	"github.com/sirupsen/logrus"
)

type accessResponse struct {
	Ok   bool        `json:"ok"`
	Data *AccessData `json:"auth"`
}

type AccessData struct {
	Id                 string
	Address            string
	IsApprovedForAll   bool
	IsUpdateManager    bool
	IsOwner            bool
	IsOperator         bool
	IsUpdateOperato    bool
	IsUpdateAuthorized bool
}

func (ad *AccessData) HasAccess() bool {
	return ad.IsUpdateAuthorized
}

type MapResponse struct {
	Ok   bool `json:"ok"`
	Data struct {
		Assets struct {
			Parcels []*Parcel `json:"parcels"`
			Estates []*Estate `json:"estates"`
		} `json:"assets"`
	} `json:"auth"`
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
	} `json:"auth"`
}

type Client interface {
	GetParcelAccessData(address string, x int64, y int64) (*AccessData, error)
}

type dclClient struct {
	ApiUrl string
	Agent  *metrics.Agent
}

func NewDclClient(apiUrl string, agent *metrics.Agent) Client {
	return &dclClient{apiUrl, agent}
}

// Retrieves the access auth of a address over a given accessData
func (dcl *dclClient) GetParcelAccessData(address string, x int64, y int64) (*AccessData, error) {
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

func (dcl *dclClient) doGet(url string, response interface{}) error {
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
