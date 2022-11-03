package trader

import (
	"log"

	"github.com/tradestax/traedor/auth"
	"github.com/tradestax/traedor/broker"
	"github.com/tradestax/traedor/config"
	"github.com/tradestax/traedor/datafeed"
	"github.com/tradestax/traedor/strategy"
)

type Trader struct {
	authHelper    auth.IAuthHelper
	broker        broker.IBroker
	data          []datafeed.IDatafeed
	dataChan      chan datafeed.Data
	errorChan     chan error
	indicatorChan chan strategy.Indicator
	strategy      strategy.IStrategy
	config        *config.Config
}

func NewTrader(c *config.Config) *Trader {
	ah := auth.NewAuthHelper(c)
	dc := make(chan datafeed.Data, 10)
	ec := make(chan error)
	ic := make(chan strategy.Indicator)
	dfs := make([]datafeed.IDatafeed, len(c.Datafeeds))
	for i, _ := range c.Datafeeds {
		dfs[i] = datafeed.NewDatafeed(&c.Datafeeds[i], ah, dc, ec)
	}
	return &Trader{
		authHelper:    ah,
		broker:        broker.NewBroker(&c.Broker),
		data:          dfs,
		dataChan:      dc,
		errorChan:     ec,
		indicatorChan: ic,
		strategy:      strategy.NewStrategy(c.Strategy, ic),
		config:        c,
	}
}

func (t *Trader) Run() {
	if err := t.authHelper.Authenticate(); err != nil {
		panic(err)
	}
	for _, df := range t.data {
		df.Start()
	}
	for {
		select {
		case err := <-t.errorChan:
			log.Println(err)
			return
		case newData := <-t.dataChan:
			t.strategy.AddData(newData)
		case newInd := <-t.indicatorChan:
			trade := broker.Trade{
				Symbol: t.config.Broker.Symbol,
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
				log.Println(err)
			}
		}
	}
}

func (t *Trader) Summary() {
	account, _ := t.broker.GetAccountStats()
	finalBalance := account.Balance()
	percentChange := (finalBalance - t.config.Broker.StartingBalance) / t.config.Broker.StartingBalance
	log.Printf("Final Balance: %.2f\n", finalBalance)
	log.Printf("Percent Change: %.2f%%\n", percentChange*100)
}
