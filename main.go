package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"github.com/decentraland/content-service/handlers"
	cid "github.com/ipfs/go-cid"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/ipsn/go-ipfs/core"
	mh "github.com/multiformats/go-multihash"
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

	router := GetRouter(config, client, ipfsNode)

	serverURL := getServerURL(config.Server.URL, config.Server.Port)
	log.Fatal(http.ListenAndServe(serverURL, router))
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
