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
	config Config
}

type Config struct {
	symbol string
}

func NewTrader() *Trader {
	return &Trader{
		broker: broker.NewBroker(),
		data: datafeed.NewDatafeed(datafeed.Config{
			Symbol:   "sin",
			Interval: "1ms",
		}),
		strategy: strategy.NewStrategy(),
		config: Config{
			symbol: "sin",
		},
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
			trade := broker.Trade{
				Symbol: t.config.symbol,
			}
			switch newInd.Direction {
			case strategy.Close:
				trade.Operation = broker.Close
			case strategy.Buy:
				trade.Operation = broker.Buy
			case strategy.Sell:
				trade.Operation = broker.Sell
			default:
				trade.Operation = broker.None
			}
			err := t.broker.SendTrade(trade)
			if err != nil {
				fmt.Println("Error on broker send")
			}
		}
	}
}
