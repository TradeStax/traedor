package config

import (
	"github.com/spf13/viper"
	"github.com/tradestax/traedor/pkg/auth"
	"github.com/tradestax/traedor/pkg/broker"
	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/strategy/types"
)

const (
	configFilename = "config"
	configType     = "yaml"
	configPath     = "."
)

type Config struct {
	AuthConfig auth.Config
	Broker     broker.Config
	Datafeeds  []datafeed.Config
	Strategy   []types.Config
	Database   DatabaseConfig
	API        APIConfig
	Workers    WorkersConfig
}

type DatabaseConfig struct {
	ConnectionString string
	MaxConnections   int
	MaxIdleTime      string
}

type APIConfig struct {
	Host string
	Port int
}

type WorkersConfig struct {
	Count int
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("Broker.StartingBalance", 100.0)
	v.SetDefault("Database.MaxConnections", 100)
	v.SetDefault("Database.MaxIdleTime", "30m")
	v.SetDefault("API.Host", "localhost")
	v.SetDefault("API.Port", 8080)
	v.SetDefault("Workers.Count", 2)
}

func readConfig(v *viper.Viper) {
	v.SetConfigName(configFilename)
	v.SetConfigType(configType)
	v.AddConfigPath(configPath)
	if err := v.ReadInConfig(); err != nil {
		panic(err)
	}
}

func New() *Config {
	v := viper.New()
	setDefaults(v)
	readConfig(v)
	var c Config
	if err := v.Unmarshal(&c); err != nil {
		panic(err)
	}
	return &c
}
