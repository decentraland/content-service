package routes

import (
	"fmt"

	"github.com/decentraland/content-service/internal/ipfs"
	"github.com/decentraland/content-service/utils"

	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/internal/handlers"
	"github.com/decentraland/content-service/metrics"
	"github.com/decentraland/content-service/storage"
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
	RpcClient        *rpc.RPC
	Filter           utils.ContentTypeFilter
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
		c.Agent, c.ParcelSizeLimit, c.RpcClient, c.Log)

	uploadHandler := handlers.NewUploadHandler(validation.NewValidator(), uploadService, c.Agent, c.Filter,
		c.ParcelAssetLimit, c.RequestTTL, c.Log)

	router.OPTIONS("/mappings", dclgin.PrefligthChecksMiddleware("GET, POST",
		fmt.Sprintf("x-upload-origin, %s", dclgin.BasicHeaders)))
	router.OPTIONS("/scenes", dclgin.PrefligthChecksMiddleware("GET", dclgin.BasicHeaders))
	router.OPTIONS("/parcel_info", dclgin.PrefligthChecksMiddleware("GET", dclgin.BasicHeaders))
	router.OPTIONS("/contents/:cid", dclgin.PrefligthChecksMiddleware("GET", dclgin.BasicHeaders))
	router.OPTIONS("/validate", dclgin.PrefligthChecksMiddleware("GET", dclgin.BasicHeaders))
	router.OPTIONS("/content/status", dclgin.PrefligthChecksMiddleware("POST", dclgin.BasicHeaders))

	router.GET("/scenes", mappingsHandler.GetScenes)
	router.GET("/parcel_info", mappingsHandler.GetInfo)
	router.GET("/contents/:cid", contentHandler.GetContents)
	router.GET("/validate", metadataHandler.GetParcelMetadata)
	router.POST("/content/status", contentHandler.CheckContentStatus)
	router.POST("/mappings", uploadHandler.UploadContent)

	dclgin.RegisterVersionEndpoint(router)

	c.Log.Debug("... Route initialization done.")
}
