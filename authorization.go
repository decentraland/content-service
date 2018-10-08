package main

import (
	"strconv"
	"strings"
)

func userCanModify(pubkey string, scene *scene) (bool, error) {
	parcels, err := getParcels(scene.Scene.Parcels)
	if err != nil {
		return false, err
	}

	for _, parcel := range parcels {
		match, err := canModify(pubkey, parcel)
		if err != nil || !match {
			return false, err
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

func canModify(pubkey string, parcel *parcel) (bool, error) {
	if pubkey == parcel.Owner {
		return true, nil
	} else if pubkey == parcel.UpdateOperator {
		return true, nil
	} else if parcel.EstateID != "" {
		estateID, err := strconv.Atoi(parcel.EstateID)
		if err != nil {
			return false, err
		}

		estate, err := getEstate(estateID)
		if err != nil {
			return false, err
		}

		if pubkey == estate.Owner {
			return true, nil
		} else if pubkey == estate.UpdateOperator {
			return true, nil
		}
	}

	return false, nil
}
