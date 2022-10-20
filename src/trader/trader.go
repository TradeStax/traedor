package trader

import (
	"fmt"

	"github.com/tradestax/traedor/broker"
	"github.com/tradestax/traedor/config"
	"github.com/tradestax/traedor/datafeed"
	"github.com/tradestax/traedor/strategy"
)

type Trader struct {
	broker   broker.IBroker
	data     datafeed.IDatafeed
	strategy strategy.IStrategy
	config   config.Config
}

func NewTrader(c config.Config) *Trader {
	return &Trader{
		broker:   broker.NewBroker(),
		data:     datafeed.NewDatafeed(c),
		strategy: strategy.NewStrategy(),
		config:   c,
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
				Symbol: t.config.Symbol,
			}
			switch newInd.Direction {
			case strategy.Close:
				trade.Operation = broker.Close
				trade.Price = newInd.Price
			case strategy.Buy:
				trade.Operation = broker.Buy
				trade.Price = newInd.Price
			case strategy.Sell:
				trade.Operation = broker.Sell
				trade.Price = newInd.Price
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

func (t *Trader) Summary() {
	account, _ := t.broker.GetAccountStats()
	finalBalance := account.Balance()
	percentChange := (finalBalance - t.config.StartingBalance) / t.config.StartingBalance
	fmt.Printf("Final Balance: %.2f\n", finalBalance)
	fmt.Printf("Percent Change: %.2f%%\n", percentChange*100)
}
