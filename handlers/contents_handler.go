package handlers

import (
	"errors"
	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/validation"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"

	"github.com/decentraland/content-service/storage"
	"github.com/gorilla/mux"
)

type GetContentCtx struct {
	Storage storage.Storage
}

func GetContent(ctx interface{}, w http.ResponseWriter, r *http.Request) error {
	c, ok := ctx.(GetContentCtx)
	if !ok {
		log.Fatal("Invalid Handler configuration")
		return NewInternalError("Invalid Configuration")
	}
	params := mux.Vars(r)

	storeValue := c.Storage.GetFile(params["cid"])

	switch c.Storage.(type) {
	case *storage.S3:
		w.Header().Add("Cache-Control", "max-age:31536000, public")
		http.Redirect(w, r, storeValue, 301)
	case *storage.Local:
		if _, err := os.Stat(storeValue); err == nil {
			w.Header().Add("Content-Disposition", "Attachment")
			http.ServeFile(w, r, storeValue)
		} else {
			return NewNotFoundError(storeValue)
		}
	default:
		return NewInternalError("Storage has unregistered type")
	}
	return nil
}

type ContentStatusCtx struct {
	Validator validation.Validator
	Service   ContentService
}

type ContentService interface {
	CheckContentStatus(content []string) (map[string]bool, error)
}

type ContentServiceImpl struct {
	RedisClient data.RedisClient
	Storage     storage.Storage
}

type ContentStatusRequest struct {
	Content []string `json:"content" validate:"required"`
}

func ContentStatus(ctx interface{}, r *http.Request) (Response, error) {
	c, ok := ctx.(ContentStatusCtx)
	if !ok {
		log.Fatal("Invalid Handler configuration")
		return nil, NewInternalError("Invalid Configuration")
	}

	var content ContentStatusRequest

	if err := ExtractContentFormJsonRequest(r, &content, c.Validator); err != nil {
		return nil, err
	}

	resp, err := c.Service.CheckContentStatus(content.Content)
	if err != nil {
		return nil, err
	}

	return NewOkJsonResponse(resp), nil
}

func (s *ContentServiceImpl) CheckContentStatus(content []string) (map[string]bool, error) {
	resp := make(map[string]bool)
	for _, cid := range content {
		uploaded, err := s.RedisClient.IsContentMember(cid)
		if err != nil {
			return nil, WrapInInternalError(err)
		}

		if !uploaded {
			if uploaded, err = s.checkContentInStorage(cid); err != nil {
				return nil, WrapInInternalError(err)
			}
		}
		resp[cid] = uploaded
	}
	return resp, nil
}

func (s *ContentServiceImpl) checkContentInStorage(cid string) (bool, error) {
	_, err := s.Storage.FileSize(cid)
	if err != nil {
		switch e := err.(type) {
		case storage.NotFoundError:
			return false, nil
		default:
			log.Infof("Unexpected error: %s", e.Error())
			return false, errors.New("unexpected error")
		}
	}
	if err = s.RedisClient.AddCID(cid); err != nil {
		log.Infof("Unexpected error: %s", err.Error())
		return false, errors.New("unexpected error")
	}
	return true, nil
}
