package main

import "github.com/tradestax/traedor/trader"

func main() {
	trader := trader.NewTrader()
	trader.Run()
	trader.Summary()
}
