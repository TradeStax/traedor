package trader

import (
	"log"
	"time"

	"github.com/tradestax/traedor/internal/config"
	"github.com/tradestax/traedor/pkg/auth"
	"github.com/tradestax/traedor/pkg/broker"
	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/storage"
	"github.com/tradestax/traedor/pkg/strategy"
	"github.com/tradestax/traedor/pkg/strategy/types"
)

type Trader struct {
	authHelper    auth.IAuthHelper
	broker        broker.IBroker
	data          []datafeed.IDatafeed
	dataChan      chan datafeed.Data
	errorChan     chan error
	indicatorChan chan types.Indicator
	strategy      types.IStrategy
	config        *config.Config
	storage       storage.IStorage
	runID         string
}

func NewTrader(c *config.Config) *Trader {
	return NewTraderWithStorage(c, nil, "")
}

func NewTraderWithStorage(c *config.Config, store storage.IStorage, runID string) *Trader {
	ah := auth.NewAuthHelper(&c.AuthConfig)
	dc := make(chan datafeed.Data, 1)
	ec := make(chan error)
	ic := make(chan types.Indicator, 1)
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
		storage:       store,
		runID:         runID,
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
			// Save tick data if storage is available
			if t.storage != nil && t.runID != "" {
				go func() {
					if err := t.storage.SaveTickData([]datafeed.Data{newData}); err != nil {
						log.Printf("Failed to save tick data: %v", err)
					}
				}()
			}
			
			t.broker.AddData(newData)
			err := t.strategy.AddData(newData)
			if err != nil {
				log.Println(err.Error())
				return
			}
			newInd := <-t.indicatorChan
			
			// Save signal if storage is available
			if t.storage != nil && t.runID != "" {
				go func() {
					signal := storage.Signal{
						RunID:     t.runID,
						Time:      time.Unix(newData.Date/1000, 0),
						Symbol:    t.config.Broker.Symbol.Name,
						Direction: newInd,
						Price:     newData.Close,
					}
					if err := t.storage.SaveSignal(t.runID, signal); err != nil {
						log.Printf("Failed to save signal: %v", err)
					}
				}()
			}
			
			//case newInd := <-t.indicatorChan:
			trade := broker.Trade{
				Symbol: t.config.Broker.Symbol.Name,
				Time:   newData.Date,
				Price:  newData.Close,
			}
			switch newInd.Direction {
			case types.Close:
				trade.Operation = broker.Close
			case types.Buy:
				trade.Operation = broker.Buy
			case types.Sell:
				trade.Operation = broker.Sell
			default:
				trade.Operation = broker.None
			}
			err = t.broker.SendTrade(trade)
			if err != nil {
				log.Fatalf("%v\n", err.Error())
			}
		}
	}
}

func (t *Trader) Summary() {
	t.broker.Summary()
	account, _ := t.broker.GetAccountStats()
	finalBalance := account.Balance()
	percentChange := (finalBalance - t.config.Broker.StartingBalance) / t.config.Broker.StartingBalance
	log.Printf("Final Balance: %.2f\n", finalBalance)
	log.Printf("Percent Change: %.2f%%\n", percentChange*100)
	
	// Save performance metrics if storage is available
	if t.storage != nil && t.runID != "" {
		// Get trades from broker
		trades, err := t.broker.GetTrades()
		if err != nil {
			log.Printf("Failed to get trades from broker: %v", err)
		} else {
			// Save all trades to storage
			for _, trade := range trades {
				if err := t.storage.SaveTrade(t.runID, trade); err != nil {
					log.Printf("Failed to save trade: %v", err)
				}
			}
		}

		// Calculate performance metrics
		metrics := t.calculatePerformanceMetrics(trades, finalBalance, percentChange)
		
		if err := t.storage.UpdateRunStatus(t.runID, storage.RunStatusCompleted, metrics); err != nil {
			log.Printf("Failed to save performance metrics: %v", err)
		}
	}
}

func (t *Trader) calculatePerformanceMetrics(trades []*broker.Trade, finalBalance, percentChange float64) *storage.PerformanceMetrics {
	if len(trades) == 0 {
		return &storage.PerformanceMetrics{
			TotalTrades:      0,
			WinningTrades:    0,
			LosingTrades:     0,
			TotalProfit:      finalBalance - t.config.Broker.StartingBalance,
			MaxDrawdown:      0.0,
			SharpeRatio:      0.0,
			WinRate:          0.0,
			AverageWin:       0.0,
			AverageLoss:      0.0,
			ProfitFactor:     0.0,
			FinalBalance:     finalBalance,
			ReturnPercentage: percentChange * 100,
		}
	}

	totalTrades := len(trades)
	winningTrades := 0
	losingTrades := 0
	totalWinAmount := 0.0
	totalLossAmount := 0.0
	maxDrawdown := 0.0

	for _, trade := range trades {
		if trade.Net > 0 {
			winningTrades++
			totalWinAmount += trade.Net
		} else if trade.Net < 0 {
			losingTrades++
			totalLossAmount += trade.Net
		}
		
		if trade.MaxDrawdown < maxDrawdown {
			maxDrawdown = trade.MaxDrawdown
		}
	}

	winRate := 0.0
	if totalTrades > 0 {
		winRate = float64(winningTrades) / float64(totalTrades) * 100
	}

	averageWin := 0.0
	if winningTrades > 0 {
		averageWin = totalWinAmount / float64(winningTrades)
	}

	averageLoss := 0.0
	if losingTrades > 0 {
		averageLoss = totalLossAmount / float64(losingTrades)
	}

	profitFactor := 0.0
	if totalLossAmount != 0 {
		profitFactor = totalWinAmount / (-totalLossAmount)
	}

	return &storage.PerformanceMetrics{
		TotalTrades:      totalTrades,
		WinningTrades:    winningTrades,
		LosingTrades:     losingTrades,
		TotalProfit:      finalBalance - t.config.Broker.StartingBalance,
		MaxDrawdown:      maxDrawdown,
		SharpeRatio:      0.0, // TODO: Calculate Sharpe ratio
		WinRate:          winRate,
		AverageWin:       averageWin,
		AverageLoss:      averageLoss,
		ProfitFactor:     profitFactor,
		FinalBalance:     finalBalance,
		ReturnPercentage: percentChange * 100,
	}
}
