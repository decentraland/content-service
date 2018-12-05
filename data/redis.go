package data

import (
	"strconv"

	"github.com/go-redis/redis"
)

type RedisClient interface {
	GetParcelMetadata(parcelID string) (map[string]interface{}, error)
	GetParcelContent(parcelID string) (map[string]string, error)
	StoreContent(key string, field string, value string) error
	StoreMetadata(key string, fields map[string]interface{}) error
	SetKey(key string, value interface{}) error
	AddCID(cid string) error
	IsContentMember(value string) (bool, error)
}

type Redis struct {
	Client *redis.Client
}

const uploadedElementsKey = "uploaded-content"
const metadataKeyPrefix = "metadata:"
const contentKeyPrefix = "content:"

func NewRedisClient(address string, password string, db int) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     address,
		Password: password,
		DB:       db,
	})

	err := client.Set("key", "value", 0).Err()

	return &Redis{client}, err
}

func (redis Redis) GetParcelMetadata(parcelID string) (map[string]interface{}, error) {

	parcelMeta, err := getParcelInformationFromCollection(redis.Client, parcelID, metadataKeyPrefix)
	if err != nil {
		return nil, err
	}

	metadata := make(map[string]interface{})
	for key, value := range parcelMeta {
		if key == "validityType" || key == "sequence" {
			intValue, err := strconv.Atoi(value)
			if err != nil {
				return nil, err
			}
			metadata[key] = intValue
		} else {
			metadata[key] = value
		}
	}
	return metadata, nil
}

func (redis Redis) GetParcelContent(parcelID string) (map[string]string, error) {
	return getParcelInformationFromCollection(redis.Client, parcelID, contentKeyPrefix)
}

func (redis Redis) StoreContent(key string, field string, value string) error {
	return redis.Client.HSet(contentKeyPrefix+key, field, value).Err()
}

func (redis Redis) StoreMetadata(key string, fields map[string]interface{}) error {
	return redis.Client.HMSet(metadataKeyPrefix+key, fields).Err()
}

func (redis Redis) SetKey(key string, value interface{}) error {
	return redis.Client.Set(key, value, 0).Err()
}

func (redis Redis) AddCID(cid string) error {
	return redis.Client.SAdd(uploadedElementsKey, cid).Err()
}

func (redis Redis) IsContentMember(value string) (bool, error) {
	res := redis.Client.SIsMember(uploadedElementsKey, value)
	if err := res.Err(); err != nil {
		return false, err
	}
	return res.Val(), nil
}

func getParcelInformationFromCollection(client *redis.Client, parcelID string, keyPrefix string) (map[string]string, error) {
	parcelCID, err := client.Get(parcelID).Result()
	if err != nil {
		return nil, err
	}

	parcelContent, err := client.HGetAll(keyPrefix + parcelCID).Result()
	if err != nil {
		return nil, err
	}

	return parcelContent, nil
}
