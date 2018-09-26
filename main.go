package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

var localStorage, s3Storage bool
var localStorageDir string

func main() {
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

	r.HandleFunc("/mappings", mappingsHandler).Methods("GET").Queries("x1", "{x1}", "y1", "{y1}", "x2", "{x2}", "y2", "{y2}")
	r.HandleFunc("/mappings", uploadHandler).Methods("POST")
	r.HandleFunc("/contents/{cid}", contentsHandler).Methods("GET")
	r.HandleFunc("/validate", validateHandler).Methods("GET").Queries("x", "{x}", "y", "{y}")

	log.Fatal(http.ListenAndServe(":8000", r))
}
