package main

import (
	"bytes"
	"encoding/xml"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/ipsn/go-ipfs/core"
)

var server *httptest.Server

func TestMain(m *testing.M) {
	// Start server
	config := GetConfig()

	client, err := initRedisClient(config)
	if err != nil {
		log.Fatal(err)
	}

	var ipfsNode *core.IpfsNode
	ipfsNode, err = initIpfsNode()
	if err != nil {
		log.Fatal(err)
	}

	router := GetRouter(config, client, ipfsNode)
	server = httptest.NewServer(router)
	defer server.Close()

	// Run tests
	code := m.Run()
	os.Exit(code)
}

type Link struct {
	A    xml.Name `xml:"a"`
	Href string   `xml:"href,attr"`
}

func TestContentsHandler(t *testing.T) {
	const CID = "123456789"
	req, err := http.NewRequest("GET", server.URL+"/contents/"+CID, nil)
	if err != nil {
		t.Fatal(err)
	}

	client := server.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	response, err2 := client.Do(req)
	if err2 != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusMovedPermanently {
		t.Error("Contents handler should respond with status code 301. Recieved code: ", response.StatusCode)
	}

	body, err3 := ioutil.ReadAll(response.Body)
	if err3 != nil {
		t.Error("Error reading body of response")
		return
	}

	link := new(Link)
	err4 := xml.NewDecoder(bytes.NewBuffer(body)).Decode(link)
	if err4 != nil {
		t.Error("Error parsing response body")
		return
	}

	expected := "https://content-service.s3.amazonaws.com/" + CID
	if link.Href != expected {
		t.Errorf("Should redirect to %s. Recieved link to : %s", expected, link.Href)
	}
}
