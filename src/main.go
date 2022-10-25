package main

import (
	"github.com/tradestax/traedor/config"
	"github.com/tradestax/traedor/trader"
)

/*
	Some current options
	Datafeed: "Generated" or "CSV"
		When using Generated
			* Symbol can be "ones" or "sin"
			* Interval determines sleep in between values
		When using CSV
			* DataPath must be specified, file to use
			* Interval determines sleep in between values
*/

func main() {
	traderConfig := config.Config{
		AuthConfig: config.AuthConfig{
			AuthHelper:  "TDA",
			UserEnvVar:  "TDAMERITRADE_CLIENT_ID",
			CallbackURL: "https://127.0.0.1/callback",
		},
		Datafeed:        "TDA",
		DataPath:        "./data/SPY_5min_sample.csv",
		Interval:        "1ms",
		StartingBalance: 100.0,
		Symbol:          "sin",
	}
	trader := trader.NewTrader(traderConfig)
	trader.Run()
	trader.Summary()
}
