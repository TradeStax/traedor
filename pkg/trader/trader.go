package trader

import (
	"fmt"
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

func (t *Trader) Run() error {
	// Update run status to running
	if t.storage != nil && t.runID != "" {
		if err := t.storage.UpdateRunStatus(t.runID, storage.RunStatusRunning, nil); err != nil {
			log.Printf("Failed to update run status to running: %v", err)
		}
	}

	if err := t.authHelper.Authenticate(); err != nil {
		t.updateRunStatusFailed(err)
		return fmt.Errorf("authentication failed: %w", err)
	}
	
	for _, df := range t.data {
		df.Start()
	}
	
	for {
		select {
		case err := <-t.errorChan:
			log.Printf("Datafeed error: %v", err)
			t.updateRunStatusFailed(err)
			return fmt.Errorf("datafeed error: %w", err)
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
				log.Printf("Strategy error: %v", err)
				t.updateRunStatusFailed(err)
				return fmt.Errorf("strategy error: %w", err)
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
				log.Printf("Broker error: %v", err)
				t.updateRunStatusFailed(err)
				return fmt.Errorf("broker error: %w", err)
			}
		}
	}
}

func (t *Trader) updateRunStatusFailed(err error) {
	if t.storage != nil && t.runID != "" {
		// You might want to store the error message in storage as well
		if updateErr := t.storage.UpdateRunStatus(t.runID, storage.RunStatusFailed, nil); updateErr != nil {
			log.Printf("Failed to update run status to failed: %v", updateErr)
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
	// Get balance history and max drawdown from broker
	balanceHistory := t.broker.GetBalanceHistory()
	maxDrawdownValue := t.broker.GetMaxDrawdown()
	maxDrawdownPercent := 0.0
	if t.config.Broker.StartingBalance > 0 {
		maxDrawdownPercent = (maxDrawdownValue / t.config.Broker.StartingBalance) * 100
	}
	
	// Convert broker balance points to storage balance points
	storageBalanceHistory := make([]storage.BalancePoint, len(balanceHistory))
	for i, bp := range balanceHistory {
		storageBalanceHistory[i] = storage.BalancePoint{
			Time:    bp.Time,
			Balance: bp.Balance,
		}
	}
	
	// Calculate drawdown history
	drawdownHistory := t.calculateDrawdownHistory(storageBalanceHistory)
	
	if len(trades) == 0 {
		return &storage.PerformanceMetrics{
			TotalTrades:      0,
			WinningTrades:    0,
			LosingTrades:     0,
			TotalProfit:      finalBalance - t.config.Broker.StartingBalance,
			MaxDrawdown:      maxDrawdownValue,
			MaxDrawdownPercent: maxDrawdownPercent,
			SharpeRatio:      0.0,
			WinRate:          0.0,
			AverageWin:       0.0,
			AverageLoss:      0.0,
			ProfitFactor:     0.0,
			FinalBalance:     finalBalance,
			ReturnPercentage: percentChange * 100,
			AverageMFE:       0.0,
			AverageMFEPercent: 0.0,
			AverageMAE:       0.0,
			AverageMAEPercent: 0.0,
			BalanceHistory:   storageBalanceHistory,
			DrawdownHistory:  drawdownHistory,
		}
	}

	totalTrades := len(trades)
	winningTrades := 0
	losingTrades := 0
	totalWinAmount := 0.0
	totalLossAmount := 0.0
	totalMFE := 0.0
	totalMAE := 0.0
	totalMFEPercent := 0.0
	totalMAEPercent := 0.0

	for _, trade := range trades {
		if trade.Net > 0 {
			winningTrades++
			totalWinAmount += trade.Net
		} else if trade.Net < 0 {
			losingTrades++
			totalLossAmount += trade.Net
		}
		
		totalMFE += trade.MFE
		totalMAE += trade.MAE
		totalMFEPercent += trade.MFEPercent
		totalMAEPercent += trade.MAEPercent
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
	
	avgMFE := 0.0
	avgMAE := 0.0
	avgMFEPercent := 0.0
	avgMAEPercent := 0.0
	if totalTrades > 0 {
		avgMFE = totalMFE / float64(totalTrades)
		avgMAE = totalMAE / float64(totalTrades)
		avgMFEPercent = totalMFEPercent / float64(totalTrades)
		avgMAEPercent = totalMAEPercent / float64(totalTrades)
	}

	return &storage.PerformanceMetrics{
		TotalTrades:      totalTrades,
		WinningTrades:    winningTrades,
		LosingTrades:     losingTrades,
		TotalProfit:      finalBalance - t.config.Broker.StartingBalance,
		MaxDrawdown:      maxDrawdownValue,
		MaxDrawdownPercent: maxDrawdownPercent,
		SharpeRatio:      0.0, // TODO: Calculate Sharpe ratio
		WinRate:          winRate,
		AverageWin:       averageWin,
		AverageLoss:      averageLoss,
		ProfitFactor:     profitFactor,
		FinalBalance:     finalBalance,
		ReturnPercentage: percentChange * 100,
		AverageMFE:       avgMFE,
		AverageMFEPercent: avgMFEPercent,
		AverageMAE:       avgMAE,
		AverageMAEPercent: avgMAEPercent,
		BalanceHistory:   storageBalanceHistory,
		DrawdownHistory:  drawdownHistory,
	}
}

func (t *Trader) calculateDrawdownHistory(balanceHistory []storage.BalancePoint) []storage.DrawdownPoint {
	if len(balanceHistory) == 0 {
		return []storage.DrawdownPoint{}
	}
	
	drawdownHistory := make([]storage.DrawdownPoint, 0)
	peakBalance := balanceHistory[0].Balance
	
	for _, bp := range balanceHistory {
		if bp.Balance > peakBalance {
			peakBalance = bp.Balance
		}
		
		drawdown := peakBalance - bp.Balance
		drawdownPercent := 0.0
		if peakBalance > 0 {
			drawdownPercent = (drawdown / peakBalance) * 100
		}
		
		drawdownHistory = append(drawdownHistory, storage.DrawdownPoint{
			Time:            bp.Time,
			Drawdown:        drawdown,
			DrawdownPercent: drawdownPercent,
		})
	}
	
	return drawdownHistory
}
