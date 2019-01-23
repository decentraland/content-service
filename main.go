package main

import (
	"context"
	"fmt"
	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/metrics"
	"github.com/decentraland/content-service/routes"
	gHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"net/http"
	"strings"

	"github.com/decentraland/content-service/config"
	"github.com/decentraland/content-service/storage"

	"github.com/ipsn/go-ipfs/core"
	log "github.com/sirupsen/logrus"
)

func main() {
	configParams := config.GetConfig("config")

	initLogger(configParams)

	log.Info("Starting server")

	router := InitializeApp(configParams)

	//CORS
	corsObj := gHandlers.AllowedOrigins([]string{"*"})

	serverURL := fmt.Sprintf(":%s", configParams.Server.Port)
	log.Fatal(http.ListenAndServe(serverURL, gHandlers.CORS(corsObj)(router)))
}

func InitializeApp(config *config.Configuration) *mux.Router {
	agent, err := metrics.Make(config.Metrics)
	if err != nil {
		log.Fatal("Error initializing metrics agent")
	}

	// Initialize Redis client
	client, err := data.NewRedisClient(config.Redis.Address, config.Redis.Password, config.Redis.DB, agent)
	if err != nil {
		log.Fatal("Error initializing Redis client")
	}

	// Initialize IPFS for CID calculations
	var ipfsNode *core.IpfsNode
	ipfsNode, err = initIpfsNode()
	if err != nil {
		log.Fatal("Error initializing IPFS node")
	}

	sto := storage.NewStorage(&config.Storage, agent)

	router := routes.GetRouter(client, sto, config.DecentralandApi.LandUrl, ipfsNode, agent)

	return router
}

func initIpfsNode() (*core.IpfsNode, error) {
	ctx, _ := context.WithCancel(context.Background())
	return core.NewNode(ctx, nil)
}

func initLogger(c *config.Configuration) {
	lvl, err := log.ParseLevel(strings.ToLower(c.LogLevel))
	if err != nil {
		log.Fatalf("Invalid log level: %s", c.LogLevel)
	}
	log.SetFormatter(&log.TextFormatter{
		TimestampFormat: "2006-01-02T15:04:05.000",
		FullTimestamp:   true,
	})

	log.SetReportCaller(true)
	log.SetLevel(lvl)
	log.Infof("Log level: %s", c.LogLevel)
}
