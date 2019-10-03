package auth

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/decentraland/content-service/internal/decentraland"

	log "github.com/sirupsen/logrus"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type Authorization interface {
	UserCanModifyParcels(pubkey string, parcelsList []string) (bool, error)
	IsSignatureValid(msg, hexSignature, hexAddress string) bool
}

type AuthorizationService struct {
	dclClient decentraland.Client
}

func NewAuthorizationService(client decentraland.Client) *AuthorizationService {
	return &AuthorizationService{client}
}

func (service AuthorizationService) UserCanModifyParcels(pubkey string, parcelsList []string) (bool, error) {
	log.Debugf("Checking Address[%s] permissions", pubkey)
	if len(parcelsList) == 0 {
		return false, fmt.Errorf("There must be at least one parcel")
	}
	for _, parcelStr := range parcelsList {
		coordinates := strings.Split(parcelStr, ",")
		if len(coordinates) != 2 {
			log.Errorf("Invalid Coordinate: %s", parcelStr)
			return false, fmt.Errorf("invalid Coordinate: %s", parcelStr)
		}

		x, err := strconv.ParseInt(coordinates[0], 10, 64)
		if err != nil {
			log.WithError(err).Errorf("Invalid Coordinate: %s", coordinates[0])
			return false, err
		}
		y, err := strconv.ParseInt(coordinates[1], 10, 64)
		if err != nil {
			log.WithError(err).Errorf("Invalid Coordinate: %s", coordinates[1])
			return false, err
		}

		log.Debugf("Verifying Address [%s] permissions over Parcel[%d,%d]", pubkey, x, y)
		access, err := service.dclClient.GetParcelAccessData(pubkey, x, y)
		if err != nil {
			return false, err
		}

		if !access.HasAccess() {
			log.Debugf("Address [%s] does not have permissions over Parcel[%d,%d]", pubkey, x, y)
			return false, nil
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

	publicKey, _ := crypto.UnmarshalPubkey(publicKeyBytes)
	sigAddress := crypto.PubkeyToAddress(*publicKey)
	ownerAddress, err := hexutil.Decode(hexAddress)
	if err != nil {
		log.Errorf("Invalid address: %s", hexAddress)
		return false
	}

	return bytes.Equal(sigAddress.Bytes(), ownerAddress)
}
