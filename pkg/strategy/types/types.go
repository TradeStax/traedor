package types

import "github.com/tradestax/traedor/pkg/types"

type IStrategy interface {
	AddData(types.Data) error
	GetIndicatorFeed() chan Indicator
}

type StratBuilder func(*Config, chan Indicator) IStrategy

type Config struct {
	Type         string
	IgnoreNone   bool
	Symbol       string
	Params       Params
	SignalParams map[string]interface{}
}

type Params struct {
	DataPath string
	Values   []string
}

type Values struct {
	Studies   map[string]float64
	Timestamp int64
}

type Indicator struct {
	Direction int
	Price     float64
	Time      int64
}

const (
	None  = 0
	Close = 1
	Buy   = 2
	Sell  = 3
)
