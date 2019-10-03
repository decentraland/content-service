package handlers

import (
	"net/http"

	"github.com/decentraland/content-service/internal/decentraland"

	"github.com/decentraland/content-service/internal/utils"

	"github.com/decentraland/content-service/internal/content"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	log "github.com/sirupsen/logrus"
)

type parcelContent struct {
	ParcelID string `json:"parcel_id"`
	ID       string `json:"id"`
	Mapping  string `json:"mapping"`
}

// Logic layer

type MappingsHandler interface {
	GetScenes(c *gin.Context)
}

type mappingsHandlerImpl struct {
	Dcl     decentraland.Client
	Storage content.Repository
	Log     *log.Logger
}

func NewMappingsHandler(dcl decentraland.Client, storage content.Repository, l *log.Logger) MappingsHandler {
	return &mappingsHandlerImpl{
		Dcl:     dcl,
		Storage: storage,
		Log:     l,
	}
}

type getScenesParams struct {
	X1 *int `form:"x1" binding:"exists,min=-150,max=150"`
	Y1 *int `form:"y1" binding:"exists,min=-150,max=150"`
	X2 *int `form:"x2" binding:"exists,min=-150,max=150"`
	Y2 *int `form:"y2" binding:"exists,min=-150,max=150"`
}

func (ms *mappingsHandlerImpl) GetScenes(c *gin.Context) {
	var p getScenesParams
	err := c.ShouldBindWith(&p, binding.Query)
	if err != nil {
		println(err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid query params"})
		return
	}

	pids := utils.RectToParcels(*p.X1, *p.Y1, *p.X2, *p.Y2, 200)
	if pids == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "too many parcels requested"})
		return
	}

	//TODO: Implement read form private deploy bucket

	ret := []*parcelContent{}

	c.JSON(http.StatusOK, gin.H{"auth": ret})
}
