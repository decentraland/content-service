package handlers

import (
	"net/http"
	"path/filepath"

	"github.com/gorilla/mux"
)

type ContentsHandler struct {
	S3Storage    bool
	LocalStorage string
}

func (handler *ContentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)

	if handler.S3Storage {
		location := getFileS3(params["cid"])
		http.Redirect(w, r, location, 301)
	} else {
		location := filepath.Join(handler.LocalStorage, params["cid"])
		w.Header().Add("Content-Disposition", "Attachment")
		http.ServeFile(w, r, location)
	}
}
