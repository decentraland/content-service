package main

import (
	"context"
	"fmt"
	"github.com/decentraland/dcl-gin/pkg/dclgin"

	"github.com/gin-gonic/gin"

	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/internal/routes"
	"github.com/decentraland/content-service/metrics"

	"github.com/decentraland/content-service/config"
	"github.com/decentraland/content-service/storage"

	"github.com/ipsn/go-ipfs/core"
	log "github.com/sirupsen/logrus"
	"github.com/toorop/gin-logrus"
)

func main() {
	conf := config.GetConfig("config")

	l := newLogger()
	router := gin.New()
	router.Use(ginlogrus.Logger(l), gin.Recovery())

	if err := setLogLevel(l, conf.LogLevel); err != nil {
		l.Fatal("error setting log level")
	}

	l.Info("Starting server")

	if conf.Metrics.Enabled {
		metricsConfig := &dclgin.HttpMetricsConfig{
			TraceName:            conf.Metrics.AppName,
			AnalyticsRateEnabled: true,
		}
		if traceError := dclgin.EnableTrace(metricsConfig, router); traceError != nil {
			log.WithError(traceError).Fatal("Unable to start metrics")
		}
		defer dclgin.StopTrace()
	}

	InitializeHandler(router, conf, l)

	addr := fmt.Sprintf("%s:%d", conf.Server.Host, conf.Server.Port)
	if err := router.Run(addr); err != nil {
		log.WithError(err).Fatal("Failed to start server.")
	}
}

func InitializeHandler(r gin.IRouter, conf *config.Configuration, l *log.Logger) {
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

	routes.AddRoutes(r, &routes.Config{
		Client:  client,
		Storage: sto,
		Node:    ipfsNode,
		Agent:   agent,
		Conf:    conf,
		Log:     l,
	})
}

func initIpfsNode() (*core.IpfsNode, error) {
	ctx, _ := context.WithCancel(context.Background())
	return core.NewNode(ctx, nil)
}

func newLogger() *log.Logger {
	l := log.New()
	formatter := log.JSONFormatter{
		FieldMap: log.FieldMap{
			log.FieldKeyTime: "@timestamp",
		},
	}
	l.SetFormatter(&formatter)
	return l
}

func setLogLevel(logger *log.Logger, level string) error {
	lvl, err := log.ParseLevel(level)
	if err == nil {
		logger.SetLevel(lvl)
	}
	return err
}
