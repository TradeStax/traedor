package config

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
