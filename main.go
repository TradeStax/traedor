package main

import (
	"github.com/tradestax/traedor/internal/config"
	"github.com/tradestax/traedor/pkg/trader"
)

func main() {
	trader := trader.NewTrader(config.New())
	trader.Run()
	trader.Summary()
}
