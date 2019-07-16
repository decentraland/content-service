package handlers

import (
	"github.com/decentraland/content-service/data"
	"github.com/decentraland/content-service/storage"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"log"
	"net/http"
)

var commitHash = "Not available"

type HealthChecker struct {
	Storage storage.Storage
	Redis   data.RedisClient
	Dcl     data.Decentraland
}

func (hc *HealthChecker) Check() (bool, map[string]string) {
	logrus.Info("Checking status")
	failures := map[string]string{}

	if ok, err := hc.checkDecentralandConnection(); !ok {
		failures["DCL-API"] = err
	}

	if ok, err := hc.checkStorage(); !ok {
		failures["Storage"] = err
	}

	if ok, err := hc.checkRedis(); !ok {
		failures["DB"] = err
	}
	return len(failures) == 0, failures
}

func (hc *HealthChecker) checkDecentralandConnection() (bool, string) {
	//TODO(mmarquez): GetParcelAccessData doesn't sound like a good test endpoint
	_, err := hc.Dcl.GetParcelAccessData("0123456789012345678901234567890123456789", 0, 0)
	if err != nil {
		logrus.Infof("Failed to connect with Decentraland: %s", err.Error())
		return false, "Failed to connect with Decentraland"
	}
	return true, ""
}

func (hc *HealthChecker) checkStorage() (bool, string) {
	// The file won't exist, but the error should reflect that
	// Any other error means there is something wrong with the storage
	_, err := hc.Storage.FileSize(uuid.New().String())
	if err != nil {
		switch e := err.(type) {
		case storage.NotFoundError:
			return true, ""
		default:
			logrus.WithError(err).Errorf("error accessing storage: %s", e.Error())
			return false, "Error accessing storage"
		}
	}
	return true, ""
}

func (hc *HealthChecker) checkRedis() (bool, string) {
	_, err := hc.Redis.IsContentMember(uuid.New().String())
	if err != nil {
		logrus.WithError(err).Error("error reading redis")
		return false, "Error connecting with DB"
	}
	return true, ""
}

type CheckResponse struct {
	Version  string            `json:"version"`
	Failures map[string]string `json:"errors"`
}

func HealthCheck(ctx interface{}, r *http.Request) (Response, error) {
	hc, ok := ctx.(HealthChecker)
	if !ok {
		log.Fatal("Invalid Handler configuration")
		return nil, NewInternalError("Invalid Configuration")
	}

	ok, failures := hc.Check()

	status := http.StatusOK
	if !ok {
		status = http.StatusServiceUnavailable
	}

	return &JsonResponse{StatusCode: status, Content: &CheckResponse{Version: commitHash, Failures: failures}}, nil
}
