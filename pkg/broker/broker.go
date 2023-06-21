package broker

import (
	"fmt"
	"log"

	"github.com/tradestax/traedor/pkg/broker/stop"
)

type Config struct {
	StartingBalance    float64
	WeeklyWithdrawl    float64
	Symbol             Symbol
	Stops              []stop.Config
	Type               string
	TrailingStopAmount float64
	FeePerSide         float64
	OpenSlippage       float64
	CloseSlippage      float64
}

type Symbol struct {
	Name       string
	Margin     float64
	PointPrice float64
}

type brokerBuilder func(*Config) IBroker

var (
	baseBrokers = map[string]brokerBuilder{
		"Futures": NewFuturesBroker,
	}
	customBrokers = map[string]brokerBuilder{}
)

func NewBroker(c *Config) IBroker {
	if f, ok := baseBrokers[c.Type]; ok {
		log.Printf("Creating %v broker", c.Type)
		return f(c)
	}
	if f, ok := customBrokers[c.Type]; ok {
		log.Printf("Creating %v broker", c.Type)
		return f(c)
	}
	log.Fatalf("Error, unrecognized broker %v\n", c.Type)
	panic(fmt.Errorf("Unable to create broker"))
	return nil
}
