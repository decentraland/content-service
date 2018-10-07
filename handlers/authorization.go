package handlers

import (
	"strconv"
	"strings"
)

func userCanModify(pubkey string, scene *scene) (bool, error) {
	parcels, err := getParcels(scene.Scene.Parcels)
	if err != nil {
		return false, err
	}

	estate, err := getEstate(scene.Scene.EstateID)
	if err != nil {
		return false, err
	}

	for _, parcel := range parcels {
		if !canModify(pubkey, parcel, estate) {
			return false, nil
		}
	}

	return true, nil
}

func getParcels(parcelsList []string) ([]*parcel, error) {
	var parcels []*parcel

	for _, parcelStr := range parcelsList {
		coordinates := strings.Split(parcelStr, ",")

		x, err := strconv.ParseInt(coordinates[0], 10, 64)
		if err != nil {
			return nil, err
		}
		y, err := strconv.ParseInt(coordinates[1], 10, 64)
		if err != nil {
			return nil, err
		}

		land, err := getParcel(int(x), int(y))
		parcels = append(parcels, land)
	}

	return parcels, nil
}

func canModify(pubkey string, parcel *parcel, estate *estate) bool {
	// TODO: check if pubkey marches update operator for parcel or estate, we are waiting on Decentraland's API
	if pubkey == parcel.Owner {
		return true
	} else if parcel.EstateID == estate.ID {
		if pubkey == estate.Owner {
			return true
		}
	}

	return false
}
