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

func GetRouter(client data.RedisClient, storage storage.Storage, dclApi string, node *core.IpfsNode, agent metrics.Agent, limits config.Limits) *mux.Router {
	r := mux.NewRouter()
	setupApiInitialVersion(r, client, storage, dclApi, node, agent, limits)
	return r
}

func setupApiInitialVersion(r *mux.Router, client data.RedisClient, storage storage.Storage, dclApi string, node *core.IpfsNode, agent metrics.Agent, limits config.Limits) {
	log.Debug("Initializing routes...")
	r.Path("/mappings").
		Methods("GET").
		Queries("nw", "{x1:-?[0-9]+},{y1:-?[0-9]+}", "se", "{x2:-?[0-9]+},{y2:-?[0-9]+}").
		Handler(&handlers.ResponseHandler{Ctx: handlers.NewMappingsService(client, data.NewDclClient(dclApi, agent)), H: handlers.GetMappings, Agent: agent, Id: "GetMappings"})

	uploadCtx := handlers.UploadCtx{
		StructValidator: validation.NewValidator(),
		Service:         handlers.NewUploadService(storage, client, node, data.NewAuthorizationService(data.NewDclClient(dclApi, agent)), agent, limits.ParcelContentLimit),
		Agent:           agent,
		Limits:          limits,
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

	contentStatusCtx := handlers.ContentStatusCtx{Service: &handlers.ContentServiceImpl{RedisClient: client}, Validator: validation.NewValidator()}

	r.Path("/content/status").
		Methods("POST").
		Headers("Content-Type", "application/json").
		Handler(&handlers.ResponseHandler{Ctx: contentStatusCtx, H: handlers.ContentStatus, Agent: agent, Id: "ContentStatus"})

	log.Debug("... Route initialization done.")
}
