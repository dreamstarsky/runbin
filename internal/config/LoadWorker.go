package config

import (
	"log"

	"github.com/spf13/viper"
)

type LimitConfig struct {
	Cpu    float32
	Memory int
	Time   float32
}

type WorkerConfig struct {
	Storage StorageConfig
	Limit   LimitConfig
	Process int
	Name    string
}

func LoadWorker(configFile string) *WorkerConfig {
	v := viper.New()
	v.SetConfigFile(configFile)
	v.SetConfigType("yaml")

	v.SetDefault("limit.cpu", 1.0)
	v.SetDefault("limit.time", 10.0)
	v.SetDefault("limit.memory", 512*1024)
	v.SetDefault("process", 1)
	v.SetDefault("name", "default name")

	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("Failed to read config file: %v", err)
	}

	var cfg WorkerConfig
	if err := v.Unmarshal(&cfg); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}

	return &cfg
}
