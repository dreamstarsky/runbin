package config

import (
	"log"

	"github.com/spf13/viper"
)

type AppConfig struct {
	Env  string
	Port int
}

type DatabaseConfig struct {
	DSN string
}

type StorageConfig struct {
	Type     string
	Database DatabaseConfig
}

type ApiConfig struct {
	App     AppConfig
	Storage StorageConfig
}

func LoadApi(configFile string) *ApiConfig {
	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")
	
	// Set default values
	v.SetDefault("app.env", "debug")
	v.SetDefault("app.port", 8080)
	v.SetDefault("storage.type", "memory")

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var cfg ApiConfig
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}

	return &cfg
}
