package handlers

import (
	"strconv"

	"github.com/go-redis/redis"
)

func getParcelMetadata(client *redis.Client, parcelID string) (map[string]interface{}, error) {
	parcelCID, err := client.Get(parcelID).Result()
	if err != nil {
		return nil, err
	}

	parcelMeta, err := client.HGetAll("metadata_" + parcelCID).Result()
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

func getParcelContent(client *redis.Client, parcelID string) (map[string]string, error) {
	parcelCID, err := client.Get(parcelID).Result()
	if err != nil {
		return nil, err
	}

	parcelMeta, err := client.HGetAll("content_" + parcelCID).Result()
	if err != nil {
		return nil, err
	}

	return parcelMeta, nil
}
