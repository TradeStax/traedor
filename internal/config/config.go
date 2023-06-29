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
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("Broker.StartingBalance", 100.0)
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
