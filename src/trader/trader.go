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
			Symbol:   "sin",
			Interval: "1ms",
		}),
		strategy: strategy.NewStrategy(),
	}
}

func (t *Trader) Run() {
	dataChan := t.data.GetDatafeed()
	dErrorChan := t.data.GetErrorChan()
	indChan := t.strategy.GetIndicatorFeed()
	for {
		select {
		case err := <-dErrorChan:
			fmt.Println(err.Error())
			return
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
