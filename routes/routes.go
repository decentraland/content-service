package routes

import (
	"github.com/decentraland/content-service/config"
	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/handlers"
	"github.com/decentraland/content-service/metrics"
	"github.com/decentraland/content-service/storage"
	"github.com/decentraland/content-service/validation"
	log "github.com/sirupsen/logrus"

	"github.com/gorilla/mux"
	"github.com/ipsn/go-ipfs/core"
)

func GetRouter(client data.RedisClient, storage storage.Storage, node *core.IpfsNode, agent *metrics.Agent, conf *config.Configuration) *mux.Router {
	r := mux.NewRouter()
	setupApiInitialVersion(r, client, storage, node, agent, conf)
	return r
}

func setupApiInitialVersion(r *mux.Router, client data.RedisClient, storage storage.Storage, node *core.IpfsNode, agent *metrics.Agent, conf *config.Configuration) {
	log.Debug("Initializing routes...")
	r.Path("/mappings").
		Methods("GET").
		Queries("nw", "{x1:-?[0-9]+},{y1:-?[0-9]+}", "se", "{x2:-?[0-9]+},{y2:-?[0-9]+}").
		Handler(&handlers.ResponseHandler{Ctx: handlers.NewMappingsService(client, data.NewDclClient(conf.DecentralandApi.LandUrl, agent)), H: handlers.GetMappings, Agent: agent, Id: "GetMappings"})

	uploadCtx := handlers.UploadCtx{
		StructValidator: validation.NewValidator(),
		Service:         handlers.NewUploadService(storage, client, node, data.NewAuthorizationService(data.NewDclClient(conf.DecentralandApi.LandUrl, agent)), agent, conf.Limits.ParcelContentLimit),
		Agent:           agent,
		Filter:          handlers.NewContentTypeFilter(conf.AllowedContentTypes),
		Limits:          conf.Limits,
	}

	r.Path("/mappings").
		Methods("POST").
		Handler(&handlers.ResponseHandler{Ctx: uploadCtx, H: handlers.UploadContent, Agent: agent, Id: "UploadContent"})

	r.Path("/contents/{cid}").
		Methods("GET").
		Handler(&handlers.Handler{Ctx: handlers.GetContentCtx{Storage: storage}, H: handlers.GetContent, Agent: agent, Id: "GetContent"})

	r.Path("/validate").
		Methods("GET").
		Queries("x", "{x:-?[0-9]+}", "y", "{y:-?[0-9]+}").
		Handler(&handlers.ResponseHandler{Ctx: handlers.NewMetadataService(client), H: handlers.GetParcelMetadata, Agent: agent, Id: "ValidateParcel"})

	contentStatusCtx := handlers.ContentStatusCtx{Service: &handlers.ContentServiceImpl{RedisClient: client, Storage: storage}, Validator: validation.NewValidator()}
	r.Path("/content/status").
		Methods("POST").
		Headers("Content-Type", "application/json").
		Handler(&handlers.ResponseHandler{Ctx: contentStatusCtx, H: handlers.ContentStatus, Agent: agent, Id: "ContentStatus"})

	checker := handlers.HealthChecker{Storage: storage, Redis: client, Dcl: data.NewDclClient(conf.DecentralandApi.LandUrl, agent)}
	r.Path("/check").
		Methods("GET").
		Handler(&handlers.ResponseHandler{Ctx: checker, H: handlers.HealthCheck, Agent: agent, Id: "HealthCheck"})

	log.Debug("... Route initialization done.")
}
