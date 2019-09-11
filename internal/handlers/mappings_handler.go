package handlers

import (
	"net/http"
	"strings"

	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/storage"
	. "github.com/decentraland/content-service/utils"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-redis/redis"
	log "github.com/sirupsen/logrus"
)

type ParcelContent struct {
	ParcelID  string            `json:"parcel_id"`
	Contents  []*ContentElement `json:"contents"`
	RootCID   string            `json:"root_cid"`
	Publisher string            `json:"publisher"`
}

type SceneContent struct {
	RootCID  string         `json:"root_cid"`
	SceneCID string         `json:"scene_cid"`
	Content  *ParcelContent `json:"content"`
}

type Scene struct {
	ParcelId string `json:"parcel_id"`
	RootCID  string `json:"root_cid"`
	SceneCID string `json:"scene_cid"`
}

type ContentElement struct {
	File string `json:"file"`
	Cid  string `json:"hash"`
}

type StringPair struct {
	A string
	B string
}

// Logic layer

type MappingsHandler interface {
	GetScenes(c *gin.Context)
	GetParcelInformation(parcelId string) (*ParcelContent, error)
	GetInfo(c *gin.Context)
}

type mappingsHandlerImpl struct {
	RedisClient data.RedisClient
	Dcl         data.Decentraland
	Storage     storage.Storage
	Log         *log.Logger
}

func NewMappingsHandler(client data.RedisClient, dcl data.Decentraland, storage storage.Storage, l *log.Logger) MappingsHandler {
	return &mappingsHandlerImpl{
		RedisClient: client,
		Dcl:         dcl,
		Storage:     storage,
		Log:         l,
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

	pids := RectToParcels(*p.X1, *p.Y1, *p.X2, *p.Y2, 200)
	if pids == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "too many parcels requested"})
		return
	}

	cids := make(map[string]bool, len(pids))

	for _, pid := range pids {
		cid, err := ms.RedisClient.GetParcelCID(pid)
		if cid == "" {
			continue
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "unexpected error, try again later"})
			return
		}

		validParcel, err := ms.RedisClient.ProcessedParcel(pid)
		if err != nil {
			log.Errorf("error when checking validity of parcel %s", pid)
			// skip on error
		}
		if !validParcel {
			continue
		}

		cids[cid] = true
	}

	ret := make([]*Scene, 0, len(cids))
	for cid, _ := range cids {
		parcels, err := ms.RedisClient.GetSceneParcels(cid)

		if err != nil && err != redis.Nil {
			ms.Log.WithError(err).Error("error reading scene from redis")
			_ = c.Error(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "unexpected error, try again later"})
			return
		}
		sceneCID, err := ms.RedisClient.GetSceneCid(cid)
		if err != nil && err != redis.Nil {
			log.Errorf("error reading scene cid for cid %s", cid)
			// we just use the empty string in this case
		}

		for _, p := range parcels {
			ret = append(ret, &Scene{
				SceneCID: sceneCID,
				RootCID:  cid,
				ParcelId: p,
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"data": ret})
}

/**
Retrieves the consolidated information of a given Parcel <ParcelContent>
if the parcel does not exists, the ParcelContent.Contents will be nil
*/
func (ms *mappingsHandlerImpl) GetParcelInformation(parcelId string) (*ParcelContent, error) {
	content, err := ms.RedisClient.GetParcelContent(parcelId)
	if content == nil || err != nil {
		return nil, err
	}

	var elements []*ContentElement

	for name, cid := range content {
		elements = append(elements, &ContentElement{File: name, Cid: cid})
	}

	metadata, err := ms.RedisClient.GetParcelMetadata(parcelId)
	if metadata == nil || err != nil {
		return nil, err
	}
	return &ParcelContent{ParcelID: parcelId, Contents: elements, RootCID: metadata["root_cid"].(string), Publisher: metadata["pubkey"].(string)}, nil
}

func (ms *mappingsHandlerImpl) GetInfo(c *gin.Context) {

	cidsParam := c.Query("cids")
	if len(cidsParam) <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid params"})
		return
	}
	cids := strings.Split(cidsParam, ",")

	parcels := make(map[string]*StringPair, len(cids))
	for _, cid := range cids {
		ps, err := ms.RedisClient.GetSceneParcels(cid)
		if err != nil && err != redis.Nil {
			ms.Log.WithError(err).Error("error reading scene from redis")
			_ = c.Error(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "unexpected error, try again later"})
			return
		}
		sceneCID := ""
		rootCID := ""
		if ps == nil || len(ps) == 0 {
			// Maybe the parameter is not the root cid, but the scene cid, which we will be eventually support better
			rootCID, err = ms.RedisClient.GetRootCid(cid)
			if err != nil && err != redis.Nil {
				log.Errorf("error when getting rootcid for hash %s with error %s", cid, err)
				continue
			}
			ps, err = ms.RedisClient.GetSceneParcels(rootCID)
			if err != nil && err != redis.Nil {
				log.Errorf("error when reading parcels for cid %s with error %s", rootCID, err)
				continue
			}
			if ps == nil || len(ps) == 0 {
				continue
			}
			sceneCID = cid
		}

		if sceneCID == "" {
			sceneCID, _ = ms.RedisClient.GetSceneCid(cid)
			rootCID = cid
		}

		parcels[rootCID] = &StringPair{A: ps[0], B: sceneCID}
	}

	ret := make([]*SceneContent, 0, len(cids))
	for k, v := range parcels {
		content, err := ms.GetParcelInformation(v.A)
		if err != nil {
			log.Errorf("error getting information for parcel %s with error %s", v.A, err)
			continue
		}

		ret = append(ret, &SceneContent{
			RootCID:  k,
			SceneCID: v.B,
			Content:  content,
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": ret})
}
