package routes

import (
	"fmt"

	"github.com/decentraland/content-service/internal/deployment"

	"github.com/decentraland/content-service/internal/ipfs"
	"github.com/decentraland/content-service/utils"

	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/internal/handlers"
	"github.com/decentraland/content-service/internal/storage"
	"github.com/decentraland/content-service/metrics"
	"github.com/decentraland/content-service/utils/rpc"
	"github.com/decentraland/content-service/validation"
	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"

	"github.com/decentraland/dcl-gin/pkg/dclgin"
)

type Config struct {
	Storage          storage.Storage
	Ipfs             *ipfs.IpfsHelper
	Agent            *metrics.Agent
	Log              *log.Logger
	DclClient        data.Decentraland
	RpcClient        rpc.RPC
	Filter           utils.ContentTypeFilter
	MRepo            deployment.Repository
	ParcelSizeLimit  int64
	ParcelAssetLimit int
	RequestTTL       int64
}

func AddRoutes(router gin.IRouter, c *Config) {
	c.Log.Debug("Initializing routes...")

	router.Use(dclgin.CorsMiddleware())

	mappingsHandler := handlers.NewMappingsHandler(c.DclClient, c.Storage, c.Log)
	contentHandler := handlers.NewContentHandler(c.Storage, c.Log)
	metadataHandler := handlers.NewMetadataHandler(c.Log)

	uploadService := handlers.NewUploadService(c.Storage, c.Ipfs, data.NewAuthorizationService(c.DclClient),
		c.Agent, c.ParcelSizeLimit, c.MRepo, c.RpcClient, c.Log)

	uploadHandler := handlers.NewUploadHandler(validation.NewValidator(), uploadService, c.Agent, c.Filter,
		c.ParcelAssetLimit, c.RequestTTL, c.Log)

	api := router.Group("/api")
	v1 := api.Group("/v1")

	v1.OPTIONS("/contents", dclgin.PrefligthChecksMiddleware("POST",
		fmt.Sprintf("x-upload-origin, %s", dclgin.BasicHeaders)))
	v1.OPTIONS("/scenes", dclgin.PrefligthChecksMiddleware("GET", dclgin.BasicHeaders))
	v1.OPTIONS("/parcel_info", dclgin.PrefligthChecksMiddleware("GET", dclgin.BasicHeaders))
	v1.OPTIONS("/contents/:cid", dclgin.PrefligthChecksMiddleware("GET", dclgin.BasicHeaders))
	v1.OPTIONS("/validate", dclgin.PrefligthChecksMiddleware("GET", dclgin.BasicHeaders))
	v1.OPTIONS("/asset_status", dclgin.PrefligthChecksMiddleware("POST", dclgin.BasicHeaders))

	v1.GET("/scenes", mappingsHandler.GetScenes)
	v1.GET("/parcel_info", mappingsHandler.GetInfo)
	v1.GET("/contents/:cid", contentHandler.GetContents)
	v1.GET("/validate", metadataHandler.GetParcelMetadata)
	v1.POST("/asset_status", contentHandler.CheckContentStatus)
	v1.POST("/contents", uploadHandler.UploadContent)

	dclgin.RegisterVersionEndpoint(router)

	c.Log.Debug("... Route initialization done.")
}
