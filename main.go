package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/decentraland/content-service/handlers"
	"github.com/decentraland/content-service/storage"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/ipsn/go-ipfs/core"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	config := GetConfig("config")

	// Initialize Redis client
	client, err := initRedisClient(config)
	if err != nil {
		log.Fatal("Error initializing Redis client")
	}

	// Initialize IPFS for CID calculations
	var ipfsNode *core.IpfsNode
	ipfsNode, err = initIpfsNode()
	if err != nil {
		log.Fatal("Error initializing IPFS node")
	}

	storage := initStorage(config)

	router := GetRouter(config, client, ipfsNode, storage)

	serverURL := getServerURL(config.Server.URL, config.Server.Port)
	log.Fatal(http.ListenAndServe(serverURL, router))
}

func initStorage(config *Configuration) storage.Storage {
	if config.S3Storage.Bucket != "" {
		return storage.NewS3(config.S3Storage.Bucket, config.S3Storage.ACL, config.S3Storage.URL)
	} else {
		sto, err := storage.NewLocal(config.LocalStorage)
		if err != nil {
			log.Fatal(err)
		}
		return sto
	}
}

func initRedisClient(config *Configuration) (*redis.Client, error) {
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

func getServerURL(serverURL string, port string) string {
	serverString := fmt.Sprintf("%s:%s", serverURL, port)
	baseURL, err := url.Parse(serverString)
	if err != nil {
		log.Fatalf("Cannot parse server url: %s", serverString)
	}
	return baseURL.Host
}


func GetRouter(config *Configuration, client *redis.Client, node *core.IpfsNode, storage storage.Storage) *mux.Router {
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
