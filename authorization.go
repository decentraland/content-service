package main

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
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

func isSignatureValid(msg, hexSignature, hexAddress string) (bool, error) {
	// Add prefix to signature message: https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_sign
	msgWithPrefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(msg), msg)
	msgHash := crypto.Keccak256Hash([]byte(msgWithPrefix))

	sigBytes, err := hexutil.Decode(hexSignature)
	if err != nil {
		return false, err
	}

	// It appears go-ethereum lib hasn't updated to accept
	// [27, 28] values, it only accepts [0, 1]
	if sigBytes[64] == 27 || sigBytes[64] == 28 {
		sigBytes[64] -= 27
	}

	publicKeyBytes, err := crypto.Ecrecover(msgHash.Bytes(), sigBytes)
	if err != nil {
		return false, err
	}

	verified := crypto.VerifySignature(publicKeyBytes, msgHash.Bytes(), sigBytes[:64])

	publicKey, err := crypto.UnmarshalPubkey(publicKeyBytes)
	sigAddress := crypto.PubkeyToAddress(*publicKey)
	ownerAddress, err := hexutil.Decode(hexAddress)
	if err != nil {
		return false, err
	}

	return verified && bytes.Equal(sigAddress.Bytes(), ownerAddress), nil
}
