package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/decentraland/content-service/handlers"
	"github.com/ipsn/go-ipfs/core"
)

var server *httptest.Server

func TestMain(m *testing.M) {
	// Start server
	config := GetConfig()

	redisClient, err := initRedisClient(config)
	if err != nil {
		log.Fatal(err)
	}

	var ipfsNode *core.IpfsNode
	ipfsNode, err = initIpfsNode()
	if err != nil {
		log.Fatal(err)
	}

	router := GetRouter(config, redisClient, ipfsNode)
	server = httptest.NewServer(router)
	defer server.Close()

	// Run tests
	code := m.Run()
	os.Exit(code)
}

func getHttpClient() *http.Client {
	// Configure http.Client to avoid following redirects
	client := server.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return client
}

type Link struct {
	A    xml.Name `xml:"a"`
	Href string   `xml:"href,attr"`
}

func TestContentsHandler(t *testing.T) {
	const CID = "123456789"

	client := getHttpClient()
	response, err := client.Get(server.URL + "/contents/" + CID)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusMovedPermanently {
		t.Error("Contents handler should respond with status code 301. Recieved code: ", response.StatusCode)
	}

	// body, err2 := ioutil.ReadAll(response.Body)
	// if err2 != nil {
	// 	t.Error("Error reading body of response")
	// 	return
	// }

	link := new(Link)
	err3 := xml.NewDecoder(response.Body).Decode(link)
	if err3 != nil {
		t.Error("Error parsing response body")
		return
	}

	expected := "https://content-service.s3.amazonaws.com/" + CID
	if link.Href != expected {
		t.Errorf("Should redirect to %s. Recieved link to : %s", expected, link.Href)
	}
}

func TestInvalidCoordinates(t *testing.T) {
	x1, y1, x2, y2 := -999, 999, -999, 999
	query := fmt.Sprintf("/mappings?nw=%d,%d&se=%d,%d", x1, y1, x2, y2)

	client := getHttpClient()
	response, err := client.Get(server.URL + query)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Error("Mappings handler should respond with status code 200. Recieved code: ", response.StatusCode)
	}

	if contentType := response.Header.Get("Content-Type"); contentType != "application/json" {
		t.Error("Mappings handler should return JSON file. Got 'Content-Type' :", contentType)
	}

	body, err2 := ioutil.ReadAll(response.Body)
	if err2 != nil {
		t.Error(err2)
	}
	bodyString := string(body)
	if bodyString != "{}" {
		t.Errorf("Mappings handler should return empty JSON when requesting invalid coordinates.\nRecieved:\n%s", bodyString)
	}
}

func TestValidCoordinates(t *testing.T) {
	x1, y1, x2, y2 := -10, 10, 10, -10
	query := fmt.Sprintf("/mappings?nw=%d,%d&se=%d,%d", x1, y1, x2, y2)

	client := getHttpClient()
	response, err := client.Get(server.URL + query)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		t.Error("Mappings handler should respond with status code 200. Recieved code: ", response.StatusCode)
	}

	if contentType := response.Header.Get("Content-Type"); contentType != "application/json" {
		t.Error("Mappings handler should return JSON file. Got 'Content-Type' :", contentType)
	}

	var jsonResponse handlers.MapResponse
	json.NewDecoder(response.Body).Decode(&jsonResponse)

	t.Error(strconv.FormatBool(jsonResponse.Ok))
	// if jsonResponse.Ok {
	// 	t.Errorf("Recieved invalid response when requesting valid map coordinates")
	// }
}

// func TestValidCoordinates(t *testing.T) {
// 	x1, y1, x2, y2 := -10, 10, 10, -10
// 	query := fmt.Sprintf("/mappings?nw=%d,%d&se=%d,%d", x1, y1, x2, y2)

// 	client := getHttpClient()
// 	response, err := client.Get(server.URL + query)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer response.Body.Close()

// 	if response.StatusCode != http.StatusOK {
// 		t.Error("Mappings handler should respond with status code 200. Recieved code: ", response.StatusCode)
// 	}

// 	if contentType := response.Header.Get("Content-Type"); contentType != "application/json" {
// 		t.Error("Mappings handler should return JSON file. Got 'Content-Type' :", contentType)
// 	}

// 	var jsonResponse handlers.MapResponse
// 	json.NewDecoder(response.Body).Decode(&jsonResponse)

// 	t.Error(strconv.FormatBool(jsonResponse.Ok))
// 	// if jsonResponse.Ok {
// 	// 	t.Errorf("Recieved invalid response when requesting valid map coordinates")
// 	// }
// }
