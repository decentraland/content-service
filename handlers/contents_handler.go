package handlers

import (
	"net/http"

	"github.com/decentraland/content-service/storage"
	"github.com/gorilla/mux"
)

type GetContentCtx struct {
	Storage storage.Storage
}

func GetContent(ctx interface{}, w http.ResponseWriter, r *http.Request) error {
	c, ok := ctx.(GetContentCtx)
	if !ok {
		return NewInternalError("Invalid Configuration")
	}
	params := mux.Vars(r)

	storeValue := c.Storage.GetFile(params["cid"])

	switch c.Storage.(type) {
	case *storage.S3:
		w.Header().Add("Cache-Control", "max-age:31536000, public")
		http.Redirect(w, r, storeValue, 301)
	case *storage.Local:
		w.Header().Add("Content-Disposition", "Attachment")
		http.ServeFile(w, r, storeValue)
	default:
		return NewInternalError("Storage has unregistered type")
	}
	return nil
}
