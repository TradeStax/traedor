package broker

import (
	"fmt"
	"log"
	"time"

	"github.com/tradestax/traedor/pkg/broker/profit"
	"github.com/tradestax/traedor/pkg/broker/stop"
)

type Config struct {
	StartingBalance    float64
	WeeklyWithdrawl    float64
	BlackoutTimes      BlackoutTimesStrings
	Symbol             Symbol
	Stops              []stop.Config
	Profits            []profit.Config
	TradeQuantity      int
	Type               string
	TrailingStopAmount float64
	FeePerSide         float64
	OpenSlippage       float64
	CloseSlippage      float64
}

type BlackoutTimesStrings struct {
	StartTime string
	EndTime   string
	TimeZone  string
}

type BlackoutTimes struct {
	StartTime time.Time
	EndTime   time.Time
	TimeZone  *time.Location
}

type Symbol struct {
	Name       string
	Margin     float64
	PointPrice float64
}

type brokerBuilder func(*Config) IBroker

const (
	timeLayout = "15:05"
)

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

func isBlackout(now, start, end time.Time) bool {
	if now.Hour() == start.Hour() {
		if now.Minute() < start.Minute() {
			return false
		} else {
			return true
		}
	} else if now.Hour() == end.Hour() {
		if now.Minute() > end.Minute() {
			return false
		} else {
			return true
		}
	} else if now.Hour() > start.Hour() && now.Hour() < end.Hour() {
		return true
	} else {
		return false
	}
}
