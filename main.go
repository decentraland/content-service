package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/ipfs/go-cid"
	mh "github.com/multiformats/go-multihash"
)

var localStorage, s3Storage bool
var localStorageDir string
var client *redis.Client
var cidPref cid.Prefix

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	// redis connection example
	client = redis.NewClient(&redis.Options{
		Addr:     "content_service_redis:6379",
		Password: "",
		DB:       0,
	})

	err := client.Set("key", "value", 0).Err()
	if err != nil {
		panic(err)
	}
	// CID creation example
	cidPref = cid.Prefix{
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

	flag.BoolVar(&localStorage, "local", false, "Local storage")
	flag.StringVar(&localStorageDir, "local-dir", "/tmp/", "Local storage directory")
	flag.BoolVar(&s3Storage, "s3", false, "S3 storage")
	flag.Parse()

	if !localStorage && !s3Storage {
		localStorage = true
	} else if localStorage && s3Storage {
		fmt.Println("You must set only ONE storage")
		os.Exit(1)
	}

	if localStorageDir[len(localStorageDir)-1:] != "/" {
		localStorageDir = localStorageDir + "/"
	}

	r := mux.NewRouter()

	r.HandleFunc("/mappings", mappingsHandler).Methods("GET").Queries("nw", "{x1},{y1}", "se", "{x2},{y2}")
	r.HandleFunc("/mappings", uploadHandler).Methods("POST")
	r.HandleFunc("/contents/{cid}", contentsHandler).Methods("GET")
	r.HandleFunc("/validate", validateHandler).Methods("GET").Queries("x", "{x}", "y", "{y}")

	log.Fatal(http.ListenAndServe(":8000", r))
}
