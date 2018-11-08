package main

import (
	"context"
	"fmt"
	"github.com/decentraland/content-service/data"
	gHandlers "github.com/gorilla/handlers"
	"log"
	"net/http"

	"github.com/decentraland/content-service/config"
	"github.com/decentraland/content-service/handlers"
	"github.com/decentraland/content-service/storage"

	"github.com/gorilla/mux"
	"github.com/ipsn/go-ipfs/core"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	configParams := config.GetConfig("config")

	// Initialize Redis client
	client, err := data.NewRedisClient(configParams.Redis.Address, configParams.Redis.Password, configParams.Redis.DB)
	if err != nil {
		log.Fatal("Error initializing Redis client")
	}

	// Initialize IPFS for CID calculations
	var ipfsNode *core.IpfsNode
	ipfsNode, err = initIpfsNode()
	if err != nil {
		log.Fatal("Error initializing IPFS node")
	}

	sto := storage.NewStorage(configParams)

	router := GetRouter(configParams, client, ipfsNode, sto)

	//CORS
	corsObj := gHandlers.AllowedOrigins([]string{"*"})

	serverURL := fmt.Sprintf(":%s", configParams.Server.Port)
	log.Fatal(http.ListenAndServe(serverURL, gHandlers.CORS(corsObj)(router)))
}

func initIpfsNode() (*core.IpfsNode, error) {
	ctx, _ := context.WithCancel(context.Background())

	return core.NewNode(ctx, nil)
}

func GetRouter(config *config.Configuration, client data.RedisClient, node *core.IpfsNode, storage storage.Storage) *mux.Router {
	r := mux.NewRouter()

	dclApi := config.DecentralandApi.LandUrl

	r.Path("/mappings").
		Methods("GET").
		Queries("nw", "{x1:-?[0-9]+},{y1:-?[0-9]+}", "se", "{x2:-?[0-9]+},{y2:-?[0-9]+}").
		Handler(&handlers.MappingsHandler{RedisClient: client, Dcl: data.NewDclClient(dclApi)})

	uploadHandler := handlers.UploadHandler{
		Storage:     storage,
		RedisClient: client,
		IpfsNode:    node,
		Auth:        data.NewAuthorizationService(data.NewDclClient(dclApi)),
	}

	r.Path("/mappings").
		Methods("POST").
		Handler(&uploadHandler)

	contentsHandler := handlers.ContentsHandler{
		Storage: storage,
	}

	r.Path("/contents/{cid}").Methods("GET").Handler(&contentsHandler)

	r.Path("/validate").
		Methods("GET").
		Queries("x", "{x:-?[0-9]+}", "y", "{y:-?[0-9]+}").
		Handler(&handlers.ValidateHandler{RedisClient: client})

	return r
}
