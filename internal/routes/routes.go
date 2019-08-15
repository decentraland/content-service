package routes

import (
	"fmt"

	"github.com/decentraland/content-service/config"
	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/internal/handlers"
	"github.com/decentraland/content-service/metrics"
	"github.com/decentraland/content-service/storage"
	"github.com/decentraland/content-service/utils/rpc"
	"github.com/decentraland/content-service/validation"
	log "github.com/sirupsen/logrus"

	"github.com/gin-gonic/gin"

	"github.com/ipsn/go-ipfs/core"

	"github.com/decentraland/dcl-gin/pkg/dclgin"
)

type Config struct {
	Client  data.RedisClient
	Storage storage.Storage
	Node    *core.IpfsNode
	Agent   *metrics.Agent
	Conf    *config.Configuration
	Log     *log.Logger
}

func AddRoutes(router gin.IRouter, c *Config) {
	c.Log.Debug("Initializing routes...")

	mappingsHandler := handlers.NewMappingsHandler(c.Client, data.NewDclClient(c.Conf.DecentralandApi.LandUrl, c.Agent), c.Storage, c.Log)
	contentHandler := handlers.NewContentHandler(c.Storage, c.Client, c.Log)
	metadataHandler := handlers.NewMetadataHandler(c.Client, c.Log)

	uploadService := handlers.NewUploadService(c.Storage, c.Client, c.Node,
		data.NewAuthorizationService(data.NewDclClient(c.Conf.DecentralandApi.LandUrl, c.Agent)),
		c.Agent, c.Conf.Limits.ParcelSizeLimit, c.Conf.Workdir, rpc.NewRPC(c.Conf.RPCConnection.URL), c.Log)

	uploadHandler := handlers.NewUploadHandler(validation.NewValidator(), uploadService, c.Agent,
		handlers.NewContentTypeFilter(c.Conf.AllowedContentTypes), c.Conf.Limits, c.Conf.UploadRequestTTL, c.Log)

	router.OPTIONS("/mappings", dclgin.PrefligthChecksMiddleware("GET, POST",
		fmt.Sprintf("x-upload-origin, %s", dclgin.BasicHeaders)))
	router.OPTIONS("/scenes", dclgin.PrefligthChecksMiddleware("GET", dclgin.BasicHeaders))
	router.OPTIONS("/parcel_info", dclgin.PrefligthChecksMiddleware("GET", dclgin.BasicHeaders))
	router.OPTIONS("/contents/:cid", dclgin.PrefligthChecksMiddleware("GET", dclgin.BasicHeaders))
	router.OPTIONS("/validate", dclgin.PrefligthChecksMiddleware("GET", dclgin.BasicHeaders))
	router.OPTIONS("/content/status", dclgin.PrefligthChecksMiddleware("POST", dclgin.BasicHeaders))

	router.GET("/mappings", mappingsHandler.GetMappings)
	router.GET("/scenes", mappingsHandler.GetScenes)
	router.GET("/parcel_info", mappingsHandler.GetInfo)
	router.GET("/contents/:cid", contentHandler.GetContents)
	router.GET("/validate", metadataHandler.GetParcelMetadata)
	router.POST("/content/status", contentHandler.CheckContentStatus)
	router.POST("/mappings", uploadHandler.UploadContent)

	dclgin.RegisterVersionEndpoint(router)

	c.Log.Debug("... Route initialization done.")
}
