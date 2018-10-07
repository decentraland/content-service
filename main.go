package main

import (
	"context"
	"fmt"
	"log"
	"net/http"

	"github.com/decentraland/content-service/handlers"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/ipfs/go-cid"
	"github.com/ipsn/go-ipfs/core"
	mh "github.com/multiformats/go-multihash"
)

func main() {
	config := GetConfig()

	// redis connection example
	client := redis.NewClient(&redis.Options{
		Addr:     config.Redis.Address,
		Password: config.Redis.Password,
		DB:       config.Redis.DB,
	})

	err := client.Set("key", "value", 0).Err()
	if err != nil {
		panic(err)
	}

	// Initialize IPFS for CID calculations
	ctx, _ := context.WithCancel(context.Background())

	var ipfsNode *core.IpfsNode
	ipfsNode, err = core.NewNode(ctx, nil)
	if err != nil {
		panic(err)
	}

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

	// flag.BoolVar(&localStorage, "local", false, "Local storage")
	// flag.StringVar(&localStorageDir, "local-dir", "/tmp/", "Local storage directory")
	// flag.BoolVar(&s3Storage, "s3", false, "S3 storage")
	// flag.Parse()

	router := GetRouter(config, client, ipfsNode)
	log.Fatal(http.ListenAndServe(":8000", router))
}

func GetRouter(config *Configuration, client *redis.Client, node *core.IpfsNode) *mux.Router {
	r := mux.NewRouter()

	r.Handle("/mappings", &handlers.MappingsHandler{RedisClient: client}).Methods("GET").Queries("nw", "{x1},{y1}", "se", "{x2},{y2}")

	uploadHandler := handlers.UploadHandler{
		S3Storage:    config.S3Storage,
		LocalStorage: config.LocalStorage,
		RedisClient:  client,
		IpfsNode:     node,
	}
	r.Handle("/mappings", &uploadHandler).Methods("POST")

	contentsHandler := handlers.ContentsHandler{
		S3Storage:    config.S3Storage,
		LocalStorage: config.LocalStorage,
	}
	r.Handle("/contents/{cid}", &contentsHandler).Methods("GET")

	r.Handle("/validate", &handlers.ValidateHandler{RedisClient: client}).Methods("GET").Queries("x", "{x}", "y", "{y}")

	return r
}
