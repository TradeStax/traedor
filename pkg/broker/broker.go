package broker

import (
	"fmt"
	"log"

	"github.com/tradestax/traedor/internal/config"
)

type brokerBuilder func(*config.BrokerConfig) IBroker

var (
	baseBrokers = map[string]brokerBuilder{
		"Futures": NewFuturesBroker,
	}
	customBrokers = map[string]brokerBuilder{}
)

func NewBroker(c *config.BrokerConfig) IBroker {
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
