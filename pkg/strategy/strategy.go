package strategy

import (
	"fmt"
	"log"

	"github.com/tradestax/traedor/pkg/strategy/emacrossover"
	"github.com/tradestax/traedor/pkg/strategy/types"
)

var (
	baseStrategies = map[string]types.StratBuilder{
		"SMA":       NewSmaStrategy,
		"MACD":      NewMacdStrategy,
		"RSI":       NewRsiStrategy,
		"Cache":     NewCacheStrategy,
		"Label":     NewLabelStrategy,
		"SC":        NewScStrategy,
		"EMA-Cross": emacrossover.NewEmaCrossoverStrategy,
	}
	customStrategies = map[string]types.StratBuilder{}
)

func NewStrategy(c []types.Config, ic chan types.Indicator) types.IStrategy {
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
