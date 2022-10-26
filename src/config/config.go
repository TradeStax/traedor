package config

import (
	"github.com/spf13/viper"
)

const (
	configFilename = "config"
	configType     = "yaml"
	configPath     = "."
)

type Config struct {
	AuthConfig      AuthConfig
	Datafeed        string
	DataPath        string
	Interval        string
	Symbol          string
	StartingBalance float64
}

type AuthConfig struct {
	AuthHelper  string
	UserEnvVar  string
	CallbackURL string
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("StartingBalance", 100.0)
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
