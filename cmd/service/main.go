package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/decentraland/content-service/internal/deployment"

	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/internal/ipfs"
	"github.com/decentraland/content-service/utils"
	"github.com/decentraland/content-service/utils/rpc"
	"github.com/decentraland/dcl-gin/pkg/dclgin"

	"github.com/gin-gonic/gin"

	"github.com/decentraland/content-service/internal/routes"
	"github.com/decentraland/content-service/metrics"
	"github.com/decentraland/dcl-viper/pkg/config"

	"github.com/decentraland/content-service/internal/storage"

	"github.com/ipsn/go-ipfs/core"
	log "github.com/sirupsen/logrus"
	ginlogrus "github.com/toorop/gin-logrus"
)

// Configuration holds global config parameters
type Configuration struct {
	Server struct {
		Port int    `overwrite-env:"SERVER_PORT" validate:"required"`
		Host string `overwrite-env:"SERVER_HOST" validate:"required"`
	}

	Storage struct {
		Bucket string `overwrite-env:"AWS_S3_BUCKET"`
		ACL    string `overwrite-env:"AWS_S3_ACL"`
		URL    string `overwrite-env:"AWS_S3_URL"`
	}

	Deployment struct {
		Bucket string `overwrite-env:"MAPPINGS_BUCKET"`
		ACL    string `overwrite-env:"MAPPINGS_URL"`
		URL    string `overwrite-env:"MAPPINGS_ACL"`
	}

	DclApi string `overwrite-env:"DCL_API" validate:"required"`

	LogLevel string `overwrite-env:"LOG_LEVEL" validate:"required"`

	Metrics struct {
		Enabled      bool   `overwrite-env:"METRICS_APP"`
		AppName      string `overwrite-env:"ANALYTICS_KEY"`
		AnalyticsKey string `overwrite-env:"METRICS_ENABLED"`
	}

	Limits struct {
		ParcelSizeLimit   int64 `overwrite-env:"LIMIT_PARCEL_SIZE" validate:"required"`
		ParcelAssetsLimit int   `overwrite-env:"LIMIT_PARCEL_ASSETS" validate:"required"`
	}

	UploadRequestTTL int64 `overwrite-env:"UPLOAD_TTL" validate:"required"`

	RPCConnection struct {
		URL string `overwrite-env:"RPCCONNECTION_URL" validate:"required"`
	}
}

func (c *Configuration) GetAllowedTypes() []string {
	var types []string
	contentEnv := os.Getenv("ALLOWED_TYPES")
	if len(contentEnv) > 0 {
		elements := strings.Split(contentEnv, ",")
		for _, t := range elements {
			types = append(types, strings.Trim(t, " "))
		}
	}
	return types
}

func main() {
	l := newLogger()
	router := gin.New()
	router.Use(ginlogrus.Logger(l), gin.Recovery())

	var conf Configuration
	if err := config.ReadConfiguration("config/config", &conf); err != nil {
		log.Fatal(err)
	}
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

	InitializeHandler(router, &conf, l)

	addr := fmt.Sprintf("%s:%d", conf.Server.Host, conf.Server.Port)
	if err := router.Run(addr); err != nil {
		log.WithError(err).Fatal("Failed to start server.")
	}
}

func InitializeHandler(r gin.IRouter, conf *Configuration, l *log.Logger) {
	agent, err := metrics.Make(metrics.Config{
		Enabled:      conf.Metrics.Enabled,
		AppName:      conf.Metrics.AppName,
		AnalyticsKey: conf.Metrics.AnalyticsKey,
	})
	if err != nil {
		log.Fatal("Error initializing metrics agent")
	}

	// Initialize IPFS for CID calculations
	var node *core.IpfsNode
	node, err = initIpfsNode()
	if err != nil {
		log.Fatal("Error initializing IPFS node")
	}

	sto := storage.NewStorage(storage.ContentBucket{
		Bucket: conf.Storage.Bucket,
		ACL:    conf.Storage.ACL,
		URL:    conf.Storage.URL,
	}, agent)

	dcl := data.NewDclClient(conf.DclApi, agent)

	rpcClient := rpc.NewRPC(conf.RPCConnection.URL)

	routes.AddRoutes(r, &routes.Config{
		Storage:          sto,
		Ipfs:             &ipfs.IpfsHelper{Node: node},
		Agent:            agent,
		Log:              l,
		DclClient:        dcl,
		RpcClient:        rpcClient,
		Filter:           utils.NewContentTypeFilter(conf.GetAllowedTypes()),
		ParcelSizeLimit:  conf.Limits.ParcelSizeLimit,
		ParcelAssetLimit: conf.Limits.ParcelAssetsLimit,
		RequestTTL:       conf.UploadRequestTTL,
		MRepo: deployment.NewRepository(&deployment.Config{
			Bucket: conf.Deployment.Bucket,
			ACL:    conf.Deployment.ACL,
			URL:    conf.Deployment.URL,
		}),
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
