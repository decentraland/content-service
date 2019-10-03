package handlers

import (
	"net/http"

	"github.com/decentraland/content-service/internal/entities"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	log "github.com/sirupsen/logrus"
)

type MetadataHandler interface {
	GetParcelMetadata(c *gin.Context)
}

func NewMetadataHandler(l *log.Logger) MetadataHandler {
	return &metadataHandlerImpl{
		Log: l,
	}
}

type metadataHandlerImpl struct {
	Log *log.Logger
}

type validateParams struct {
	X *int `form:"x" binding:"exists,min=-150,max=150"`
	Y *int `form:"y" binding:"exists,min=-150,max=150"`
}

func (mh *metadataHandlerImpl) GetParcelMetadata(c *gin.Context) {
	var p validateParams
	err := c.ShouldBindWith(&p, binding.Query)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query params"})
		return
	}
	//TODO: retrieve x,y -> CID, CID -> proof.json from private bucket

	c.JSON(http.StatusOK, entities.DeployProof{})
}
