package handlers

import (
	"strconv"

	"github.com/go-redis/redis"
)

func getParcelMetadata(client *redis.Client, parcelID string) (map[string]interface{}, error) {

	parcelMeta, err := getParcelInformationFromCollection(client, parcelID, "metadata_")
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
	return getParcelInformationFromCollection(client, parcelID, "content_")
}

func getParcelInformationFromCollection(client *redis.Client, parcelID string, collection string) (map[string]string, error) {
	parcelCID, err := client.Get(parcelID).Result()
	if err != nil {
		return nil, err
	}

	parcelContent, err := client.HGetAll(collection + parcelCID).Result()
	if err != nil {
		return nil, err
	}

	return parcelContent, nil
}
