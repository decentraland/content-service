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

	r.Handle("/mappings", &handlers.MappingsHandler{RedisClient: client, Dcl: data.NewDclClient(dclApi)}).Methods("GET").Queries("nw", "{x1},{y1}", "se", "{x2},{y2}")

	uploadHandler := handlers.UploadHandler{
		Storage:     storage,
		RedisClient: client,
		IpfsNode:    node,
		Auth:        data.NewAuthorizationService(data.NewDclClient(dclApi)),
	}
	r.Handle("/mappings", &uploadHandler).Methods("POST")

	contentsHandler := handlers.ContentsHandler{
		Storage: storage,
	}
	r.Handle("/contents/{cid}", &contentsHandler).Methods("GET")

	r.Handle("/validate", &handlers.ValidateHandler{RedisClient: client}).Methods("GET").Queries("x", "{x}", "y", "{y}")

	return r
}
