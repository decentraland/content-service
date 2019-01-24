package config

import (
	log "github.com/sirupsen/logrus"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Configuration holds global config parameters
type Configuration struct {
	Server              Server
	Storage             Storage
	Redis               Redis
	DecentralandApi     DecentralandApi
	LogLevel            string
	Metrics             NewRelic
	AllowedContentTypes []string
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

type StorageType string

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
	Port string
	URL  string
}

type NewRelic struct {
	AppName string
	AppKey  string
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

	if config.Server.URL[len(config.Server.URL)-1:] != "/" {
		config.Server.URL = config.Server.URL + "/"
	}

	return &config
}

// Read configurations from ENV to overwrite (if present) config file values
func readEnvVariables(v *viper.Viper) {
	// Server Configuration
	v.BindEnv("server.port", "SERVER_PORT")
	v.BindEnv("server.url", "SERVER_URL")
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
	v.BindEnv("metrics.appKey", "METRICS_KEY")

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
