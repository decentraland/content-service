package handlers

import (
	"fmt"

	"github.com/decentraland/content-service/internal/decentraland"

	"github.com/decentraland/content-service/internal/deployment"

	"github.com/decentraland/content-service/internal/ipfs"
	"github.com/decentraland/content-service/internal/utils"

	"github.com/decentraland/content-service/internal/auth"
	"github.com/decentraland/content-service/internal/content"
	"github.com/decentraland/content-service/internal/metrics"
	"github.com/decentraland/content-service/internal/utils/rpc"
	"github.com/decentraland/content-service/internal/validation"
	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"

	"github.com/decentraland/dcl-gin/pkg/dclgin"
)

type Config struct {
	Storage          content.Repository
	Ipfs             *ipfs.IpfsHelper
	Agent            *metrics.Agent
	Log              *log.Logger
	DclClient        decentraland.Client
	RpcClient        rpc.RPC
	Filter           utils.ContentTypeFilter
	MRepo            deployment.Repository
	ParcelSizeLimit  int64
	ParcelAssetLimit int
	RequestTTL       int64
}

func RegisterEndpoints(router gin.IRouter, c *Config) {
	c.Log.Debug("Initializing routes...")

	router.Use(dclgin.CorsMiddleware())

	mappingsHandler := NewMappingsHandler(c.DclClient, c.Storage, c.Log)
	contentHandler := NewContentHandler(c.Storage, c.Log)
	metadataHandler := NewMetadataHandler(c.Log)

	uploadService := NewUploadService(c.Storage, c.Ipfs, auth.NewAuthorizationService(c.DclClient),
		c.Agent, c.ParcelSizeLimit, c.MRepo, c.RpcClient, c.Log)

	uploadHandler := NewUploadHandler(validation.NewValidator(), uploadService, c.Agent, c.Filter,
		c.ParcelAssetLimit, c.RequestTTL, c.Log)

	api := router.Group("/api")
	v1 := api.Group("/v1")

	v1.OPTIONS("/contents", dclgin.PrefligthChecksMiddleware("POST",
		fmt.Sprintf("x-upload-origin, %s", dclgin.BasicHeaders)))

	v1.OPTIONS("/scenes", dclgin.PrefligthChecksMiddleware("GET", dclgin.BasicHeaders))
	v1.OPTIONS("/contents/:cid", dclgin.PrefligthChecksMiddleware("GET", dclgin.BasicHeaders))
	v1.OPTIONS("/validate", dclgin.PrefligthChecksMiddleware("GET", dclgin.BasicHeaders))
	v1.OPTIONS("/asset_status", dclgin.PrefligthChecksMiddleware("POST", dclgin.BasicHeaders))

	v1.GET("/scenes", mappingsHandler.GetScenes)
	v1.GET("/contents/:cid", contentHandler.GetContents)
	v1.GET("/validate", metadataHandler.GetParcelMetadata)
	v1.POST("/asset_status", contentHandler.CheckContentStatus)
	v1.POST("/contents", uploadHandler.UploadContent)

	dclgin.RegisterVersionEndpoint(router)

	c.Log.Debug("... Route initialization done.")
}
