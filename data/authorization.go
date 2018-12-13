package data

import (
	"bytes"
	"fmt"
	log "github.com/sirupsen/logrus"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type Authorization interface {
	UserCanModifyParcels(pubkey string, parcelsList []string) (bool, error)
	IsSignatureValid(msg, hexSignature, hexAddress string) bool
}

type AuthorizationService struct {
	dclClient Decentraland
}

func NewAuthorizationService(client Decentraland) *AuthorizationService {
	return &AuthorizationService{client}
}

func (service AuthorizationService) UserCanModifyParcels(pubkey string, parcelsList []string) (bool, error) {
	log.Debugf("Checking Address[%] permissions", pubkey)
	parcels, err := getParcels(parcelsList, service.dclClient)
	if err != nil {
		return false, err
	}

	for _, parcel := range parcels {
		match, err := canModify(pubkey, parcel, service.dclClient)
		if err != nil || !match {
			return false, err
		}
	}

	return true, nil
}

func (service AuthorizationService) IsSignatureValid(msg, hexSignature, hexAddress string) bool {
	log.Debugf("Validating Signature. Message[%s], Signature[%s], Address[%s]", msg, hexSignature, hexAddress)
	// Add prefix to signature message: https://github.com/ethereum/wiki/wiki/JSON-RPC#eth_sign
	msgWithPrefix := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(msg), msg)
	msgHash := crypto.Keccak256Hash([]byte(msgWithPrefix))

	sigBytes, err := hexutil.Decode(hexSignature)
	if err != nil {
		log.Errorf("Invalid message signature: %s", hexSignature)
		return false
	}

	// It appears go-ethereum lib hasn't updated to accept
	// [27, 28] values, it only accepts [0, 1]
	if sigBytes[64] == 27 || sigBytes[64] == 28 {
		sigBytes[64] -= 27
	}

	publicKeyBytes, err := crypto.Ecrecover(msgHash.Bytes(), sigBytes)
	if err != nil {
		log.Errorf("Invalid message: %s", msg)
		return false
	}

	verified := crypto.VerifySignature(publicKeyBytes, msgHash.Bytes(), sigBytes[:64])

	publicKey, _ := crypto.UnmarshalPubkey(publicKeyBytes)
	sigAddress := crypto.PubkeyToAddress(*publicKey)
	ownerAddress, err := hexutil.Decode(hexAddress)
	if err != nil {
		log.Errorf("Invalid address: %s", hexAddress)
		return false
	}

	return verified && bytes.Equal(sigAddress.Bytes(), ownerAddress)
}

func getParcels(parcelsList []string, dcl Decentraland) ([]*Parcel, error) {
	var parcels []*Parcel

	for _, parcelStr := range parcelsList {
		coordinates := strings.Split(parcelStr, ",")

		x, err := strconv.ParseInt(coordinates[0], 10, 64)
		if err != nil {
			log.Debugf("Invalid Coordinate: %s", coordinates[0])
			return nil, err
		}
		y, err := strconv.ParseInt(coordinates[1], 10, 64)
		if err != nil {
			log.Debugf("Invalid Coordinate: %s", coordinates[1])
			return nil, err
		}

		land, err := dcl.GetParcel(int(x), int(y))
		if err != nil {
			log.Errorf("Unable to retrieve parcel from DCL: %d,%d", x, y)
			return parcels, err
		}

		parcels = append(parcels, land)
	}

	return parcels, nil
}

func canModify(pubkey string, parcel *Parcel, dcl Decentraland) (bool, error) {
	log.Debugf("Verifying Address [%s] permissions over Parcel[%s,%s]", pubkey, parcel.X, parcel.Y)
	if pubkey == parcel.Owner {
		return true, nil
	} else if pubkey == parcel.UpdateOperator {
		return true, nil
	} else if parcel.EstateID != "" {
		estateID, err := strconv.Atoi(parcel.EstateID)
		if err != nil {
			log.Errorf("Invalid estate id: %s", parcel.EstateID)
			return false, err
		}

		estate, err := dcl.GetEstate(estateID)
		if err != nil {
			log.Errorf("Unable to retrieve parcel from DCL: %d", estateID)
			return false, err
		}

		if pubkey == estate.Owner {
			return true, nil
		} else if pubkey == estate.UpdateOperator {
			return true, nil
		}
	}

	log.Debugf("Address [%s] not allowed to modify Parcel[%s,%s]", pubkey, parcel.X, parcel.Y)
	return false, nil
}
