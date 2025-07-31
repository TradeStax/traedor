package strategy

import (
	"fmt"
	"log"

	"github.com/tradestax/traedor/pkg/strategy/types"
)

var (
	baseStrategies = map[string]types.StratBuilder{
		"SMA":            NewSmaStrategy,
		"SMA_CROSSOVER":  NewSmaCrossoverStrategy,
		"MACD":           NewMacdStrategy,
		"RSI":            NewRsiStrategy,
		"SignalAdapter":  NewSignalAdapterStrategyFactory,
	}
	customStrategies = map[string]types.StratBuilder{}
)

func NewStrategy(c []types.Config, ic chan types.Indicator) types.IStrategy {
	if len(c) == 0 {
		// No strategy configured - likely using signals only
		// Use SignalAdapter strategy which handles signal-based trading
		log.Printf("No strategy configured, using SignalAdapter for signal-based trading")
		signalConfig := &types.Config{
			Type: "SignalAdapter",
		}
		return NewSignalAdapterStrategyFactory(signalConfig, ic)
	} else if len(c) == 1 {
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
