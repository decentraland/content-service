package config

import (
	"os"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/spf13/viper"
)

// Configuration holds global config parameters
type Configuration struct {
	Server              Server
	Storage             Storage
	Redis               Redis
	DecentralandApi     DecentralandApi
	LogLevel            string
	Metrics             Metrics
	AllowedContentTypes []string
	Limits              Limits
	Workdir             string
	UploadRequestTTL    int64
	RPCConnection       RPCConnection
}

type DecentralandApi struct {
	LandUrl string
}

type Redis struct {
	Address  string
	Password string
	DB       int
}

type Storage struct {
	StorageType  string
	RemoteConfig RemoteStorage
	LocalPath    string
}

type Limits struct {
	ParcelSizeLimit   int64
	ParcelAssetsLimit int
}

type StorageType string

type RPCConnection struct {
	URL string
}

const (
	REMOTE StorageType = "REMOTE"
	LOCAL  StorageType = "LOCAL"
)

type RemoteStorage struct {
	Bucket string
	ACL    string
	URL    string
}

type Server struct {
	Port int
	Host string
}

type Metrics struct {
	Enabled      bool
	AppName      string
	AnalyticsKey string
}

// GetConfig populates a Configuration struct from a config file
func GetConfig(name string) *Configuration {
	var config Configuration

	viper.SetConfigName(name)
	viper.AddConfigPath(".")

	err := viper.ReadInConfig()
	if err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	readEnvVariables(viper.GetViper())

	err = viper.Unmarshal(&config)
	if err != nil {
		log.Fatalf("Unable to decode config file into struct, %s", err)
	}

	return &config
}

// Read configurations from ENV to overwrite (if present) config file values
func readEnvVariables(v *viper.Viper) {
	// Server Configuration
	v.BindEnv("server.port", "SERVER_PORT")
	v.BindEnv("server.host", "SERVER_HOST")
	// Storage Configuration
	v.BindEnv("storage.storageType", "STORAGE_TYPE")
	v.BindEnv("storage.remoteConfig.bucket", "AWS_S3_BUCKET")
	v.BindEnv("storage.remoteConfig.url", "AWS_S3_URL")
	v.BindEnv("storage.remoteConfig.acl", "AWS_S3_ACL")
	v.BindEnv("storage.localPath", "LOCAL_STORAGE_PATH")
	// Redis Configuration
	v.BindEnv("redis.address", "REDIS_ADDRESS")
	v.BindEnv("redis.password", "REDIS_PASSWORD")
	v.BindEnv("redis.db", "REDIS_DB")
	// DCL API
	v.BindEnv("decentralandapi.landurl", "DCL_API")
	// LOG LEVEL
	v.BindEnv("logLevel", "LOG_LEVEL")
	//Metrics
	v.BindEnv("metrics.appName", "METRICS_APP")
	v.BindEnv("metrics.analyticsKey", "ANALYTICS_KEY")
	v.BindEnv("metrics.enabled", "METRICS_ENABLED")

	//Limits
	v.BindEnv("limits.parcelSizeLimit", "LIMIT_PARCEL_SIZE")
	v.BindEnv("limits.parcelAssetsLimit", "LIMIT_PARCEL_ASSETS")

	v.BindEnv("workdir", "WORK_DIR")

	v.BindEnv("uploadRequestTTL", "UPLOAD_TTL")

	v.BindEnv("rpcconnection.url", "RPCCONNECTION_URL")

	//Allowed content types
	contentEnv := os.Getenv("ALLOWED_TYPES")
	if len(contentEnv) > 0 {
		elements := strings.Split(contentEnv, ",")
		var types []string
		for _, t := range elements {
			types = append(types, strings.Trim(t, " "))
		}
		v.Set("allowedContentTypes", types)
	}
}
