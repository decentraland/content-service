package handlers

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/decentraland/content-service/data"
	log "github.com/sirupsen/logrus"

	"github.com/decentraland/content-service/storage"
)

type ContentHandler interface {
	GetContents(c *gin.Context)
	CheckContentStatus(c *gin.Context)
}

type contentHandlerImpl struct {
	Storage     storage.Storage
	RedisClient data.RedisClient
	Log         *log.Logger
}

func NewContentHandler(storage storage.Storage, l *log.Logger) ContentHandler {
	return &contentHandlerImpl{
		Storage:     storage,
		RedisClient: nil,
		Log:         l,
	}
}

func (ch *contentHandlerImpl) GetContents(c *gin.Context) {
	cid := c.Param("cid")
	storeValue := ch.Storage.GetFile(cid)

	c.Writer.Header().Set("Cache-Control", "max-age:31536000, public")
	c.Redirect(http.StatusMovedPermanently, storeValue)
}

type contentStatusRequest struct {
	Content []string `json:"content" validate:"required"`
}

func (ch *contentHandlerImpl) CheckContentStatus(c *gin.Context) {
	var statusReq contentStatusRequest
	if err := c.ShouldBindJSON(&statusReq); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	resp := make(map[string]bool)
	for _, cid := range statusReq.Content {
		uploaded, err := ch.RedisClient.IsContentMember(cid)
		if err != nil {
			ch.Log.WithError(err).Error("fail to read redis")
			_ = c.Error(err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "try again later"})
			return
		}

		if !uploaded {
			if uploaded, err = ch.checkContentInStorage(cid); err != nil {
				ch.Log.WithError(err).Error("fail to check content")
				c.JSON(http.StatusInternalServerError, gin.H{"error": "try again later"})
				return
			}
		}
		resp[cid] = uploaded
	}
	c.JSON(http.StatusOK, resp)
}

func (ch *contentHandlerImpl) checkContentInStorage(cid string) (bool, error) {
	_, err := ch.Storage.FileSize(cid)
	if err != nil {
		switch e := err.(type) {
		case storage.NotFoundError:
			return false, nil
		default:
			log.WithError(err).Errorf("error while reading storage: %s", e.Error())
			return false, err
		}
	}
	if err = ch.RedisClient.AddCID(cid); err != nil {
		log.WithError(err).Error("fail to save into redis")
		return false, errors.New("unexpected error")
	}
	return true, nil
}
