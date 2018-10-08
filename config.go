package main

import (
	"log"

	"github.com/spf13/viper"
)

// Configuration holds global config parameters
type Configuration struct {
	Server struct {
		URL  string
		Port string
	}
	S3Storage    bool
	LocalStorage string

	Redis struct {
		Address  string
		Password string
		DB       int
	}
}

// GetConfig populates a Configuration struct from a config file
func GetConfig() *Configuration {
	var config Configuration

	viper.SetConfigName("config")
	viper.AddConfigPath(".")

	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}
	err := viper.Unmarshal(&config)
	if err != nil {
		log.Fatalf("Unable to decode config file into struct, %s", err)
	}

	if config.LocalStorage[len(config.LocalStorage)-1:] != "/" {
		config.LocalStorage = config.LocalStorage + "/"
	}

	return &config
}
