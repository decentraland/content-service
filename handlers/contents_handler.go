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

	storeValue := handler.Storage.GetFile(params["cid"])

	switch handler.Storage.(type) {
	case *storage.S3:
		w.Header().Add("Cache-Control", "max-age:31536000, public")
		http.Redirect(w, r, storeValue, 301)
	case *storage.Local:
		w.Header().Add("Content-Disposition", "Attachment")
		http.ServeFile(w, r, storeValue)
	default:
		handle500(w, errors.New("Storage has unregistered type"))
	}
}
