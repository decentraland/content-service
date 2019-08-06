package data

import (
	"fmt"
	"strconv"
	"time"

	"github.com/decentraland/content-service/metrics"
	"github.com/sirupsen/logrus"

	"github.com/go-redis/redis"
)

type RedisClient interface {
	GetParcelMetadata(parcelID string) (map[string]interface{}, error)
	GetParcelContent(parcelID string) (map[string]string, error)
	StoreContent(key string, field string, value string) error
	StoreMetadata(key string, fields map[string]interface{}) error
	AddCID(cid string) error
	IsContentMember(value string) (bool, error)

	// Retrieves the root cid of the given parcel
	GetParcelCID(pid string) (string, error)

	// Deletes the mapping root cid -> [parcels]
	ClearScene(cid string) error
	// Updates the mapping root cid -> [parcels]
	SetSceneParcels(cid string, pid []string) error
	// Gets the mapping root cid -> [parcels]
	GetSceneParcels(cid string) ([]string, error)

	// Flags parcels as processed, this is used for flagging when the root cid -> parcels maps has been created
	SetProcessedParcel(pid string) error
	// Retrieves whether a parcels has been processed
	ProcessedParcel(pid string) (bool, error)

	// Saves the map root cid <--> scene cid
	SaveRootCidSceneCid(rootCID, sceneCID string) error
	// Retrieves the scene cid given the root cid of a scene
	GetSceneCid(rootCID string) (string, error)
	// Retrieves the root cid given the scene cid of a scene
	GetRootCid(sceneCID string) (string, error)
}

type Redis struct {
	Client *redis.Client
	Agent  *metrics.Agent
}

const uploadedElementsKey = "uploaded-content"
const metadataKeyPrefix = "metadata_"
const contentKeyPrefix = "content_"
const proccessedSet = "processedSet"
const rootScenePrefix = "root-scene:"

func NewRedisClient(address string, password string, db int, agent *metrics.Agent) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       db,
	})
	err := client.Set("key", "value", 0).Err()
	return &Redis{Client: client, Agent: agent}, err
}

func (r Redis) GetParcelMetadata(parcelID string) (map[string]interface{}, error) {
	t := time.Now()
	parcelMeta, err := r.getParcelInformationFromCollection(parcelID, metadataKeyPrefix)
	if err != nil {
		logrus.Errorf("Redis error: %s", err.Error())
		return nil, err
	}

	if parcelMeta == nil {
		logrus.Debugf("Parcel[%s] Metadata not found", parcelID)
		return nil, nil
	}

	metadata := make(map[string]interface{})
	for key, value := range parcelMeta {
		if key == "validityType" || key == "sequence" || key == "timestamp" {
			intValue, err := strconv.Atoi(value)
			if err != nil {
				return nil, err
			}
			metadata[key] = intValue
		} else {
			metadata[key] = value
		}
	}
	r.Agent.RecordGetParcelMetadata(time.Since(t))
	return metadata, nil
}

func (r Redis) GetParcelContent(parcelID string) (map[string]string, error) {
	t := time.Now()
	res, err := r.getParcelInformationFromCollection(parcelID, contentKeyPrefix)
	r.Agent.RecordGetParcelContent(time.Since(t))
	if res == nil {
		logrus.Tracef("Parcel[%s] Content not found", parcelID)
		return nil, nil
	}
	return res, err
}

func (r Redis) StoreContent(key string, field string, value string) error {
	t := time.Now()
	res := r.Client.HSet(contentKeyPrefix+key, field, value)
	r.Agent.RecordStoreContent(time.Since(t))
	return res.Err()
}

func (r Redis) StoreMetadata(key string, fields map[string]interface{}) error {
	t := time.Now()
	res := r.Client.HMSet(metadataKeyPrefix+key, fields)
	r.Agent.RecordStoreMetadata(time.Since(t))
	return res.Err()
}

func (r Redis) setKey(key string, value interface{}) error {
	return r.Client.Set(key, value, 0).Err()
}

func (r Redis) AddCID(cid string) error {
	return r.Client.SAdd(uploadedElementsKey, cid).Err()
}

func (r Redis) IsContentMember(value string) (bool, error) {
	t := time.Now()
	res := r.Client.SIsMember(uploadedElementsKey, value)
	r.Agent.RecordIsMemberTime(time.Since(t))

	if err := res.Err(); err != nil {
		logrus.Errorf("Redis error: %s", err.Error())
		return false, err
	}
	return res.Val(), nil
}

func (r Redis) getParcelInformationFromCollection(parcelID string, keyPrefix string) (map[string]string, error) {
	parcelCID, err := r.Client.Get(parcelID).Result()

	if err == redis.Nil {
		return nil, nil
	}

	if err != nil {
		logrus.Errorf("Redis error: %s", err.Error())
		return nil, err
	}

	parcelContent, err := r.Client.HGetAll(keyPrefix + parcelCID).Result()
	if err != nil {
		logrus.Errorf("Redis error: %s", err.Error())
		return nil, err
	}

	return parcelContent, nil
}

func (r Redis) GetParcelCID(pid string) (string, error) {
	cid, err := r.Client.Get(pid).Result()
	if err != nil && err != redis.Nil {
		logrus.Errorf("Redis error: %s", err.Error())
		return "", err
	}
	return cid, nil
}

func (r Redis) ProcessedParcel(pid string) (bool, error) {
	member, err := r.Client.SIsMember(proccessedSet, pid).Result()
	if err != nil && err != redis.Nil {
		return false, err
	}
	if err == redis.Nil {
		return false, nil
	}
	return member, nil
}

func (r Redis) SetProcessedParcel(pid string) error {
	_, err := r.Client.SAdd(proccessedSet, pid).Result()
	return err
}

// This function maps scene cid to list of parcels
// "Qvcslk2duadjao0rsdfaZaAAA"... -> ["35,-145", "-22,14"]
// Every every parcel or pair of coordinates must be unique for between all scenes, so we need to check
// for every parcel if it got a scene before and delete it if needed
// Parcels will map its own cid, "99,108" -> Qvcslk2duadjao0rsdfaZaAAA...
// We can be tempted to remove the old parcel -> old cid map, but that would break the /mappings map, so
// for now we are going to keep that map and check the validity of the cid when requesting /scenes, thing that will be
// updatable once removed the /mappings endpoint
func (r Redis) SetSceneParcels(scene string, parcels []string) error {
	if len(parcels) == 0 {
		return fmt.Errorf("Trying to push empty parcels list for scene %s", scene)
	}

	// We first iterate over all parcels deleting the mappings root cid -> [parcels] for all
	// the scenes of the parcels of the parameter
	cids := make(map[string]bool, len(parcels))
	for _, p := range parcels {
		cid, err := r.GetParcelCID(p)
		if err != nil {
			return err
		}
		cids[cid] = true
	}

	for cid, _ := range cids {
		_, err := r.Client.Del(cid).Result()
		if err != nil {
			return fmt.Errorf("redis error when cleaning old scenes: %s", err)
		}
	}

	_, err := r.Client.LPush(scene, parcels).Result()
	for _, pid := range parcels {
		err := r.setKey(pid, scene)
		if err != nil {
			return fmt.Errorf("redis error when updating scene %s", err)
		}
	}

	return err
}

func (r Redis) ClearScene(cid string) error {
	_, err := r.Client.Del(cid).Result()
	return err
}

func (r Redis) GetSceneParcels(cid string) ([]string, error) {
	scenes, err := r.Client.LRange(cid, 0, -1).Result()
	if err != nil {
		return nil, err
	}
	return scenes, nil
}

func (r Redis) SaveRootCidSceneCid(rootCID, sceneCID string) error {
	_, err := r.Client.Set(rootScenePrefix+rootCID, sceneCID, 0).Result()
	if err != nil {
		return err
	}
	_, err = r.Client.Set(rootScenePrefix+sceneCID, rootCID, 0).Result()
	return err
}
func (r Redis) GetSceneCid(rootCID string) (string, error) {
	ret, err := r.Client.Get(rootScenePrefix + rootCID).Result()
	if err != nil {
		return "", err
	}
	return ret, nil
}

func (r Redis) GetRootCid(sceneCID string) (string, error) {
	ret, err := r.Client.Get(rootScenePrefix + sceneCID).Result()
	if err != nil {
		return "", err
	}
	return ret, nil
}
