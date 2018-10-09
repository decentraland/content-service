package main

func getParcelMetadata(parcelID string) (map[string]string, error) {
	parcelCID, err := client.Get(parcelID).Result()
	if err != nil {
		return nil, err
	}

	parcelMeta, err := client.HGetAll("metadata_" + parcelCID).Result()
	if err != nil {
		return nil, err
	}

	return parcelMeta, nil
}

func getParcelContent(parcelID string) (map[string]string, error) {
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
