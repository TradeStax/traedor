package trader

import (
	"fmt"

	"github.com/tradestax/traedor/broker"
	"github.com/tradestax/traedor/datafeed"
	"github.com/tradestax/traedor/strategy"
)

type Trader struct {
	broker   broker.IBroker
	data     datafeed.IDatafeed
	strategy strategy.IStrategy
}

func NewTrader() *Trader {
	return &Trader{
		broker: broker.NewBroker(),
		data: datafeed.NewDatafeed(datafeed.Config{
			Symbol:   "SPY",
			Interval: "5M",
		}),
		strategy: strategy.NewStrategy(),
	}
}

func (t *Trader) Run() {
	dataChan := t.data.GetDatafeed()
	indChan := t.strategy.GetIndicatorFeed()
	for {
		select {
		case newData := <-dataChan:
			t.strategy.AddData(newData)
		case newInd := <-indChan:
			switch newInd.Direction {
			case strategy.Close:
				fmt.Println("Close Trade")
			case strategy.Buy:
				fmt.Println("Create Buy")
			case strategy.Sell:
				fmt.Println("Create Sell")
			default:
				fmt.Println("No indicator")
			}
		}
	}
}
