package main

import (
	"context"
	"log"
	"net/http"

	"github.com/decentraland/content-service/config"
	"github.com/decentraland/content-service/handlers"
	"github.com/decentraland/content-service/storage"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/ipsn/go-ipfs/core"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	configParams := config.GetConfig("config")

	// Initialize Redis client
	client, err := initRedisClient(configParams)
	if err != nil {
		log.Fatal("Error initializing Redis client")
	}

	// Initialize IPFS for CID calculations
	var ipfsNode *core.IpfsNode
	ipfsNode, err = initIpfsNode()
	if err != nil {
		log.Fatal("Error initializing IPFS node")
	}

	sto := storage.NewStorage(configParams)

	router := GetRouter(configParams, client, ipfsNode, sto)

	serverURL := config.GetServerURL(configParams.Server.URL, configParams.Server.Port)
	log.Fatal(http.ListenAndServe(serverURL, router))
}

// func initStorage(config *config.Configuration) storage.Storage {
// 	if config.S3Storage.Bucket != "" {
// 		return storage.NewS3(config.S3Storage.Bucket, config.S3Storage.ACL, config.S3Storage.URL)
// 	} else {
// 		err := os.MkdirAll(config.LocalStorage, os.ModePerm)
// 		if err != nil {
// 			log.Fatal(err)
// 		}
// 		return storage.NewLocal(config.LocalStorage)
// 	}
// }

func initRedisClient(config *config.Configuration) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Address,
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})

	err := client.Set("key", "value", 0).Err()

	return client, err
}

func initIpfsNode() (*core.IpfsNode, error) {
	ctx, _ := context.WithCancel(context.Background())

	return core.NewNode(ctx, nil)
}

func GetRouter(config *config.Configuration, client *redis.Client, node *core.IpfsNode, storage storage.Storage) *mux.Router {
	r := mux.NewRouter()

	r.Handle("/mappings", &handlers.MappingsHandler{RedisClient: client}).Methods("GET").Queries("nw", "{x1},{y1}", "se", "{x2},{y2}")

	uploadHandler := handlers.UploadHandler{
		Storage: storage,
		RedisClient:  client,
		IpfsNode:     node,
	}
	r.Handle("/mappings", &uploadHandler).Methods("POST")

	contentsHandler := handlers.ContentsHandler{
		Storage: storage,
	}
	r.Handle("/contents/{cid}", &contentsHandler).Methods("GET")

	r.Handle("/validate", &handlers.ValidateHandler{RedisClient: client}).Methods("GET").Queries("x", "{x}", "y", "{y}")

	return r
}
