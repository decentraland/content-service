package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"

	log "github.com/sirupsen/logrus"

	"github.com/decentraland/content-service/internal/content"
)

type ContentHandler interface {
	GetContents(c *gin.Context)
	CheckContentStatus(c *gin.Context)
}

type contentHandlerImpl struct {
	Storage content.Repository
	Log     *log.Logger
}

func NewContentHandler(storage content.Repository, l *log.Logger) ContentHandler {
	return &contentHandlerImpl{
		Storage: storage,
		Log:     l,
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
		uploaded, err := ch.checkContentInStorage(cid)
		if err != nil {
			ch.Log.WithError(err).Error("fail to check content")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "try again later"})
			return
		}
		resp[cid] = uploaded
	}
	c.JSON(http.StatusOK, resp)
}

func (ch *contentHandlerImpl) checkContentInStorage(cid string) (bool, error) {
	_, err := ch.Storage.FileSize(cid)
	if err != nil {
		switch e := err.(type) {
		case content.NotFoundError:
			return false, nil
		default:
			log.WithError(err).Errorf("error while reading content: %s", e.Error())
			return false, err
		}
	}
	return true, nil
}
