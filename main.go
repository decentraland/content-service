package main

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/metrics"
	"github.com/decentraland/content-service/routes"
	gHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"github.com/decentraland/content-service/config"
	"github.com/decentraland/content-service/storage"

	"github.com/ipsn/go-ipfs/core"
	log "github.com/sirupsen/logrus"

	muxtrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gorilla/mux"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

func main() {
	conf := config.GetConfig("config")

	initLogger(conf)

	log.Info("Starting server")

	//CORS
	corsObj := gHandlers.AllowedOrigins([]string{"*"})
	headersObj := gHandlers.AllowedHeaders([]string{"*", "x-upload-origin"})

	serverURL := fmt.Sprintf(":%s", conf.Server.Port)
	handler := InitializeHandler(conf)
	if conf.Metrics.Enabled {
		defer tracer.Stop()
	}

	log.Fatal(http.ListenAndServe(serverURL, gHandlers.CORS(corsObj, headersObj)(handler)))
}

func InitializeHandler(conf *config.Configuration) http.Handler {
	agent, err := metrics.Make(conf.Metrics)
	if err != nil {
		log.Fatal("Error initializing metrics agent")
	}

	// Initialize Redis client
	client, err := data.NewRedisClient(conf.Redis.Address, conf.Redis.Password, conf.Redis.DB, agent)
	if err != nil {
		log.Fatal("Error initializing Redis client")
	}

	// Initialize IPFS for CID calculations
	var ipfsNode *core.IpfsNode
	ipfsNode, err = initIpfsNode()
	if err != nil {
		log.Fatal("Error initializing IPFS node")
	}

	sto := storage.NewStorage(&conf.Storage, agent)

	if conf.Metrics.Enabled {
		tracer.Start(tracer.WithServiceName("test-go"))
		tracer := muxtrace.NewRouter(muxtrace.WithServiceName(conf.Metrics.AppName), muxtrace.WithAnalytics(true))
		routes.AddRoutes(tracer.Router, client, sto, ipfsNode, agent, conf)
		return tracer
	} else {
		router := mux.NewRouter()
		routes.AddRoutes(router, client, sto, ipfsNode, agent, conf)
		return router
	}
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
