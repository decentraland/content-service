package config

import (
	"log"

	"github.com/spf13/viper"
)

// Configuration holds global config parameters
type Configuration struct {
	Server          Server
	S3Storage       S3Storage
	LocalStorage    string
	Redis           Redis
	DecentralandApi DecentralandApi
}

type DecentralandApi struct {
	LandUrl string
}

type Redis struct {
	Address  string
	Password string
	DB       int
}

type S3Storage struct {
	Bucket string
	ACL    string
	URL    string
}

type Server struct {
	Port string
	URL  string
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

	if config.LocalStorage[len(config.LocalStorage)-1:] != "/" {
		config.LocalStorage = config.LocalStorage + "/"
	}

	if config.Server.URL[len(config.Server.URL)-1:] != "/" {
		config.Server.URL = config.Server.URL + "/"
	}

	return &config
}

// Read configurations from ENV to overwrite (if present) config file values
func readEnvVariables(v *viper.Viper) {
	// S3 Configuration
	v.BindEnv("s3storage.bucket", "AWS_S3_BUCKET")
	v.BindEnv("s3storage.url", "AWS_S3_URL")
	v.BindEnv("s3storage.acl", "AWS_S3_ACL")
	// Redis Configuration
	v.BindEnv("redis.address", "REDIS_ADDRESS")
	v.BindEnv("redis.password", "REDIS_PASSWORD")
	v.BindEnv("redis.db", "REDIS_DB")
	// DCL API
	v.BindEnv("decentralandapi.landurl", "DCL_API")
}
