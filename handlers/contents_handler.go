package handlers

import (
	"errors"
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
		handle500(w, errors.New("Storage has unregistered type"))
	}
}
