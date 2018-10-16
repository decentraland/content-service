package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	conf "github.com/decentraland/content-service/config"
	"github.com/decentraland/content-service/handlers"
	"github.com/decentraland/content-service/storage"
	cid "github.com/ipfs/go-cid"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/ipsn/go-ipfs/core"
	mh "github.com/multiformats/go-multihash"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	config := conf.GetConfig("config")

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

	// CID creation example
	cidPref := cid.Prefix{
		Version:  1,
		Codec:    cid.Raw,
		MhType:   mh.SHA2_256,
		MhLength: -1, // default length
	}
	ci, _ := cidPref.Sum([]byte("Hello World!"))
	fmt.Println("Created CID: ", ci)

	// CID decoding coding example
	c, _ := cid.Decode("zdvgqEMYmNeH5fKciougvQcfzMcNjF3Z1tPouJ8C7pc3pe63k")
	fmt.Println("Got CID: ", c)

	router := GetRouter(config, client, ipfsNode, storage)

	serverURL := conf.GetServerURL(config.Server.URL, config.Server.Port)
	log.Fatal(http.ListenAndServe(serverURL, router))
}

func initStorage(config *conf.Configuration) storage.Storage {
	if config.S3Storage.Bucket != "" {
		return storage.NewS3(config.S3Storage.Bucket, config.S3Storage.ACL, config.S3Storage.URL)
	} else {
		return storage.NewLocal(config.LocalStorage)
	}
}

func initRedisClient(config *conf.Configuration) (*redis.Client, error) {
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

func GetRouter(config *conf.Configuration, client *redis.Client, node *core.IpfsNode, storage storage.Storage) *mux.Router {
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
