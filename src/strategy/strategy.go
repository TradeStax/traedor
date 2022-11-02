package strategy

import (
	"fmt"

	"github.com/tradestax/traedor/config"
)

func NewStrategy(c []config.StrategyConfig, ic chan Indicator) IStrategy {
	if len(c) == 1 {
		switch c[0].Type {
		case "SMA":
			fmt.Println("Creating SMA strategy")
			return NewSmaStrategy(c[0], ic)
		case "RSI":
			fmt.Println("Creating RSI strategy")
			return NewRsiStrategy(c[0], ic)
		default:
			panic(fmt.Errorf("Unrecognized Strategy Type"))
		}
	} else if len(c) > 1 {
		fmt.Println("Configuring Ensemble")
		return NewEnsembleStrategy(c, ic)
	} else {
		panic(fmt.Errorf("Unrecognized Strategy Config"))
	}
}
