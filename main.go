package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

// var localStorage, s3Storage bool
// var localStorageDir string
// var client *redis.Client

// var cidPref cid.Prefix

func GetRouter() *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/mappings", mappingsHandler).Methods("GET").Queries("nw", "{x1},{y1}", "se", "{x2},{y2}")
	router.HandleFunc("/mappings", uploadHandler).Methods("POST")
	router.HandleFunc("/contents/{cid}", contentsHandler).Methods("GET")
	router.HandleFunc("/validate", validateHandler).Methods("GET").Queries("x", "{x}", "y", "{y}")
	return router
}

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

	// if !localStorage && !s3Storage {
	// 	localStorage = true
	// } else if localStorage && s3Storage {
	// 	fmt.Println("You must set only ONE storage")
	// 	os.Exit(1)
	// }

	// if localStorageDir[len(localStorageDir)-1:] != "/" {
	// 	localStorageDir = localStorageDir + "/"
	// }

	router := GetRouter()
	log.Fatal(http.ListenAndServe(":8000", router))
}
