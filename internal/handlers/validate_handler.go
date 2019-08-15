package handlers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/decentraland/content-service/data"
	log "github.com/sirupsen/logrus"
)

type MetadataHandler interface {
	GetParcelMetadata(c *gin.Context)
}

func NewMetadataHandler(client data.RedisClient, l *log.Logger) MetadataHandler {
	return &metadataHandlerImpl{
		RedisClient: client,
		Log:         l,
	}
}

type metadataHandlerImpl struct {
	RedisClient data.RedisClient
	Log         *log.Logger
}

type validateParams struct {
	X int `form:"x" binding:"required,numeric,min=-150,max=150"`
	Y int `form:"y" binding:"required,numeric,min=-150,max=150"`
}

func (mh *metadataHandlerImpl) GetParcelMetadata(c *gin.Context) {
	var p validateParams
	err := c.ShouldBindWith(&p, binding.Query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query params"})
		return
	}
	parcelId := fmt.Sprintf("%d,%d", p.X, p.Y)
	mh.Log.Debugf("Retrieving parcel metadata. Parcel[%s]", parcelId)
	parcelMeta, err := mh.RedisClient.GetParcelMetadata(parcelId)
	if err != nil {
		mh.Log.WithError(err).Error("error reading parcel metadata from redis")
		_ = c.Error(err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "unexpected error, try again later"})
		return
	}

	if parcelMeta == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "parcel metadata not found"})
		return
	}

	c.JSON(http.StatusOK, parcelMeta)
}
