package routes

import (
	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/handlers"
	"github.com/decentraland/content-service/storage"
	"github.com/decentraland/content-service/validation"

	"github.com/gorilla/mux"
	"github.com/ipsn/go-ipfs/core"
)

func GetRouter(client data.RedisClient, storage storage.Storage, dclApi string, node *core.IpfsNode) *mux.Router {
	r := mux.NewRouter()
	setupApiInitialVersion(r, client, storage, dclApi, node)
	return r
}

func setupApiInitialVersion(r *mux.Router, client data.RedisClient, storage storage.Storage, dclApi string, node *core.IpfsNode) {
	r.Path("/mappings").
		Methods("GET").
		Queries("nw", "{x1:-?[0-9]+},{y1:-?[0-9]+}", "se", "{x2:-?[0-9]+},{y2:-?[0-9]+}").
		Handler(&handlers.ResponseHandler{Ctx: handlers.NewMappingsService(client, data.NewDclClient(dclApi)), H: handlers.GetMappings})

	uploadCtx := handlers.UploadCtx{
		StructValidator: validation.NewValidator(),
		Service:         handlers.NewUploadService(storage, client, node, data.NewAuthorizationService(data.NewDclClient(dclApi))),
	}

	r.Path("/mappings").
		Methods("POST").
		Handler(&handlers.ResponseHandler{Ctx: uploadCtx, H: handlers.UploadContent})

	r.Path("/contents/{cid}").
		Methods("GET").
		Handler(&handlers.Handler{Ctx: handlers.GetContentCtx{Storage: storage}, H: handlers.GetContent})

	r.Path("/validate").
		Methods("GET").
		Queries("x", "{x:-?[0-9]+}", "y", "{y:-?[0-9]+}").
		Handler(&handlers.ResponseHandler{Ctx: handlers.NewMetadataService(client), H: handlers.GetParcelMetadata})

	contentStatusCtx := handlers.ContentStatusCtx{Service: &handlers.ContentServiceImpl{RedisClient: client}, Validator: validation.NewValidator()}

	r.Path("/content/status").
		Methods("POST").
		Headers("Content-Type", "application/json").
		Handler(&handlers.ResponseHandler{Ctx: contentStatusCtx, H: handlers.ContentStatus})
}
