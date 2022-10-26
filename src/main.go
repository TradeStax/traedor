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
	trader := trader.NewTrader(config.New())
	trader.Run()
	trader.Summary()
}
