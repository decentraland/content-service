package main

import (
	"context"
	"fmt"
	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/validation"
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
		Handler(&handlers.ResponseHandler{Ctx: handlers.GetMappingsCtx{RedisClient: client, Dcl: data.NewDclClient(dclApi)}, H: handlers.GetMappings})

	uploadCtx := handlers.UploadCtx{
		Storage:         storage,
		RedisClient:     client,
		IpfsNode:        node,
		Auth:            data.NewAuthorizationService(data.NewDclClient(dclApi)),
		StructValidator: validation.NewValidator(),
	}

	r.Path("/mappings").
		Methods("POST").
		Handler(&handlers.ResponseHandler{Ctx: &uploadCtx, H: handlers.UploadContent})

	getContentCtx := handlers.GetContentCtx{
		Storage: storage,
	}

	r.Path("/contents/{cid}").Methods("GET").Handler(&handlers.Handler{Ctx: &getContentCtx, H: handlers.GetContent})

	r.Path("/validate").
		Methods("GET").
		Queries("x", "{x:-?[0-9]+}", "y", "{y:-?[0-9]+}").
		Handler(&handlers.ResponseHandler{Ctx: &handlers.ValidateParcelCtx{RedisClient: client}, H: handlers.GetParcelMetadata})

	return r
}
