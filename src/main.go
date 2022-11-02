package main

import (
	"github.com/tradestax/traedor/config"
	"github.com/tradestax/traedor/trader"
)

func main() {
	trader := trader.NewTrader(config.New())
	trader.Run()
	trader.Summary()
}
