package main

import (
	"context"
	"fmt"
	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/routes"
	gHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"log"
	"net/http"

	"github.com/decentraland/content-service/config"
	"github.com/decentraland/content-service/storage"

	"github.com/ipsn/go-ipfs/core"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	configParams := config.GetConfig("config")

	router := InitializeApp(configParams)

	//CORS
	corsObj := gHandlers.AllowedOrigins([]string{"*"})

	serverURL := fmt.Sprintf(":%s", configParams.Server.Port)
	log.Fatal(http.ListenAndServe(serverURL, gHandlers.CORS(corsObj)(router)))
}

func InitializeApp(config *config.Configuration) *mux.Router {
	// Initialize Redis client
	client, err := data.NewRedisClient(config.Redis.Address, config.Redis.Password, config.Redis.DB)
	if err != nil {
		log.Fatal("Error initializing Redis client")
	}

	// Initialize IPFS for CID calculations
	var ipfsNode *core.IpfsNode
	ipfsNode, err = initIpfsNode()
	if err != nil {
		log.Fatal("Error initializing IPFS node")
	}

	sto := storage.NewStorage(&config.Storage)

	router := routes.GetRouter(client, sto, config.DecentralandApi.LandUrl, ipfsNode)

	return router
}

func initIpfsNode() (*core.IpfsNode, error) {
	ctx, _ := context.WithCancel(context.Background())
	return core.NewNode(ctx, nil)
}
