package strategy

import (
	"fmt"
	"log"

	"github.com/tradestax/traedor/internal/config"
)

type stratBuilder func(config.StrategyConfig, chan Indicator) IStrategy

var (
	baseStrategies = map[string]stratBuilder{
		"SMA":   NewSmaStrategy,
		"MACD":  NewMacdStrategy,
		"RSI":   NewRsiStrategy,
		"Cache": NewCacheStrategy,
		"SC":    NewScStrategy,
	}
	customStrategies = map[string]stratBuilder{}
)

func NewStrategy(c []config.StrategyConfig, ic chan Indicator) IStrategy {
	if len(c) == 1 {
		if f, ok := baseStrategies[c[0].Type]; ok {
			log.Printf("Creating %v strategy", c[0].Type)
			return f(c[0], ic)
		}
		if f, ok := customStrategies[c[0].Type]; ok {
			log.Printf("Creating %v strategy", c[0].Type)
			return f(c[0], ic)
		}
		panic(fmt.Errorf("Unrecognized Strategy %v", c[0].Type))
	} else if len(c) > 1 {
		fmt.Println("Configuring Ensemble")
		return NewEnsembleStrategy(c, ic)
	}
	panic(fmt.Errorf("Unrecognized Strategy Config"))
}
