package strategy

import (
	"fmt"
	"log"
	"time"
)

type Config struct {
	Type   string
	Symbol string
	Params Params
}

type Params struct {
	DataPath  string
	Values    []string
	Timeframe time.Duration
}

type stratBuilder func(*Config, chan Indicator) IStrategy

var (
	baseStrategies = map[string]stratBuilder{
		"SMA":   NewSmaStrategy,
		"MACD":  NewMacdStrategy,
		"RSI":   NewRsiStrategy,
		"Cache": NewCacheStrategy,
		"Label": NewLabelStrategy,
		"SC":    NewScStrategy,
	}
	customStrategies = map[string]stratBuilder{}
)

func NewStrategy(c []Config, ic chan Indicator) IStrategy {
	if len(c) == 1 {
		if f, ok := baseStrategies[c[0].Type]; ok {
			log.Printf("Creating %v strategy", c[0].Type)
			return f(&c[0], ic)
		}
		if f, ok := customStrategies[c[0].Type]; ok {
			log.Printf("Creating %v strategy", c[0].Type)
			return f(&c[0], ic)
		}
		panic(fmt.Errorf("Unrecognized Strategy %v", c[0].Type))
	} else if len(c) > 1 {
		fmt.Println("Configuring Ensemble")
		return NewEnsembleStrategy(c, ic)
	}
	panic(fmt.Errorf("Unrecognized Strategy Config"))
}
