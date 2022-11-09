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
	AuthConfig AuthConfig
	Broker     BrokerConfig
	Datafeeds  []DatafeedConfig
	Strategy   []StrategyConfig
}

type AuthConfig struct {
	AuthHelper  string
	UserEnvVar  string
	CallbackURL string
}

type BrokerConfig struct {
	StartingBalance float64
	Symbol          string
}

type DatafeedConfig struct {
	DataPath  string
	Fields    string
	Interval  string
	Service   string
	Symbol    string
	Type      string
	StartTime string
	EndTime   string
	Print     bool
}

type StrategyConfig struct {
	Type   string
	Symbol string
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
