package handlers

import (
	"log"
	"net/http"

	"github.com/decentraland/content-service/storage"
	"github.com/gorilla/mux"
)

type ContentsHandler struct {
	Storage storage.Storage
}

func (handler *ContentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	location := handler.Storage.GetFile(params["cid"])

	switch handler.Storage.(type) {
	case *storage.S3:
		http.Redirect(w, r, location, 301)
	case *storage.Local:
		w.Header().Add("Content-Disposition", "Attachment")
		http.ServeFile(w, r, location)
	default:
		log.Println("Storage has unregistered type")
		http.Error(w, http.StatusText(500), 500)
	}
}
