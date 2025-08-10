package config

import (
	"log"
	"path/filepath"

	"github.com/spf13/viper"
)

var config *viper.Viper

// Init is an exported method that takes the environment starts the viper
// (external lib) and returns the configuration struct.
func Init(env string) {
	var err error
	config = viper.New()
	config.SetConfigType("yaml")
	config.SetConfigName("default")
	config.AddConfigPath("config/")
	err = config.ReadInConfig()
	if err != nil {
		log.Fatal("error on parsing default configuration file")
	}

	// Map environment names to config files
	configName := env
	switch env {
	case "development":
		configName = "testnet"
	case "production":
		configName = "mainnet"
	// Keep other environments as-is (e.g., "test")
	}

	envConfig := viper.New()
	envConfig.SetConfigType("yaml")
	envConfig.AddConfigPath("config/")
	envConfig.SetConfigName(configName)
	err = envConfig.ReadInConfig()
	if err != nil {
		log.Fatalf("error on parsing %s configuration file: %v", configName, err)
	}

	config.MergeConfigMap(envConfig.AllSettings())
}

func relativePath(basedir string, path *string) {
	p := *path
	if len(p) > 0 && p[0] != '/' {
		*path = filepath.Join(basedir, p)
	}
}

func GetConfig() *viper.Viper {
	return config
}
