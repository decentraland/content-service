package config

import (
	"log"

	"github.com/spf13/viper"
)

// Configuration holds global config parameters
type Configuration struct {
	Server struct {
		Port string
		URL  string
	}
	S3Storage struct {
		Bucket string
		ACL    string
		URL    string
	}
	LocalStorage string

	Redis struct {
		Address  string
		Password string
		DB       int
	}

	Decentraland struct {
		LandApi string
	}
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
