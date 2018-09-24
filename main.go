package main

import (
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

func mappingsHandler(w http.ResponseWriter, r *http.Request) {
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
}

func contentsHandler(w http.ResponseWriter, r *http.Request) {
}

func validateHandler(w http.ResponseWriter, r *http.Request) {
}

func main() {
	r := mux.NewRouter()

	r.HandleFunc("/mappings", mappingsHandler).Methods("GET").Queries("x1", "{x1}", "y1", "{y1}", "x2", "{x2}", "y2", "{y2}")
	r.HandleFunc("/mappings", uploadHandler).Methods("POST")
	r.HandleFunc("/contents/{cid}", contentsHandler).Methods("GET")
	r.HandleFunc("/validate", validateHandler).Methods("GET").Queries("x", "{x}", "y", "{y}")

	log.Fatal(http.ListenAndServe(":8000", r))
}
