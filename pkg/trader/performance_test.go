package trader

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/tradestax/traedor/pkg/broker"
	"github.com/tradestax/traedor/pkg/storage"
)

func TestCalculatePerformanceMetrics(t *testing.T) {
	// Create a test trader instance
	trader := newTestTrader()
	
	// Create sample trades with MFE/MAE data
	trades := []*broker.Trade{
		{
			Symbol:     "ES",
			Operation:  broker.Buy,
			Quantity:   1,
			OpenPrice:  4500.0,
			ClosePrice: 4510.0,
			OpenTime:   time.Now().Unix(),
			CloseTime:  time.Now().Add(time.Hour).Unix(),
			Net:        495.0, // 10 points * 50 - fees
			MFE:        1000.0, // 20 points * 50
			MAE:        250.0,  // 5 points * 50
			MFEPercent: 2.22,
			MAEPercent: 0.56,
		},
		{
			Symbol:     "ES",
			Operation:  broker.Buy,
			Quantity:   1,
			OpenPrice:  4520.0,
			ClosePrice: 4515.0,
			OpenTime:   time.Now().Add(2 * time.Hour).Unix(),
			CloseTime:  time.Now().Add(3 * time.Hour).Unix(),
			Net:        -255.0, // -5 points * 50 - fees
			MFE:        500.0,  // 10 points * 50
			MAE:        750.0,  // 15 points * 50
			MFEPercent: 1.11,
			MAEPercent: 1.66,
		},
		{
			Symbol:     "ES",
			Operation:  broker.Buy,
			Quantity:   1,
			OpenPrice:  4515.0,
			ClosePrice: 4530.0,
			OpenTime:   time.Now().Add(4 * time.Hour).Unix(),
			CloseTime:  time.Now().Add(5 * time.Hour).Unix(),
			Net:        745.0, // 15 points * 50 - fees
			MFE:        1250.0, // 25 points * 50
			MAE:        500.0,  // 10 points * 50
			MFEPercent: 2.77,
			MAEPercent: 1.11,
		},
	}
	
	finalBalance := 10985.0 // Starting balance + net profits
	percentChange := (finalBalance - 10000) / 10000 // 9.85%
	
	// Create mock broker with balance history
	trader.broker = &mockBroker{
		balanceHistory: []broker.BalancePoint{
			{Time: time.Now(), Balance: 10000},
			{Time: time.Now().Add(time.Hour), Balance: 10495},
			{Time: time.Now().Add(2 * time.Hour), Balance: 10495},
			{Time: time.Now().Add(3 * time.Hour), Balance: 10240},
			{Time: time.Now().Add(4 * time.Hour), Balance: 10240},
			{Time: time.Now().Add(5 * time.Hour), Balance: 10985},
		},
		maxDrawdown: 255.0,
	}
	
	// Trader already has config from newTestTrader()
	
	metrics := trader.calculatePerformanceMetrics(trades, finalBalance, percentChange)
	
	// Test basic metrics
	assert.Equal(t, 3, metrics.TotalTrades)
	assert.Equal(t, 2, metrics.WinningTrades)
	assert.Equal(t, 1, metrics.LosingTrades)
	assert.InDelta(t, 985.0, metrics.TotalProfit, 0.01)
	assert.Equal(t, 255.0, metrics.MaxDrawdown)
	assert.InDelta(t, 2.55, metrics.MaxDrawdownPercent, 0.01) // 255/10000 * 100
	
	// Test win rate
	assert.InDelta(t, 66.67, metrics.WinRate, 0.01) // 2/3 * 100
	
	// Test average win/loss
	assert.InDelta(t, 620.0, metrics.AverageWin, 0.01) // (495 + 745) / 2
	assert.InDelta(t, -255.0, metrics.AverageLoss, 0.01)
	
	// Test profit factor
	assert.InDelta(t, 4.86, metrics.ProfitFactor, 0.01) // 1240 / 255
	
	// Test MFE/MAE averages
	assert.InDelta(t, 916.67, metrics.AverageMFE, 0.01) // (1000 + 500 + 1250) / 3
	assert.InDelta(t, 500.0, metrics.AverageMAE, 0.01)  // (250 + 750 + 500) / 3
	assert.InDelta(t, 2.03, metrics.AverageMFEPercent, 0.01) // (2.22 + 1.11 + 2.77) / 3
	assert.InDelta(t, 1.11, metrics.AverageMAEPercent, 0.01) // (0.56 + 1.66 + 1.11) / 3
	
	// Test balance history conversion
	assert.Len(t, metrics.BalanceHistory, 6)
	assert.Equal(t, 10000.0, metrics.BalanceHistory[0].Balance)
	assert.Equal(t, 10985.0, metrics.BalanceHistory[5].Balance)
	
	// Test drawdown history calculation
	assert.Len(t, metrics.DrawdownHistory, 6)
	assert.Equal(t, 0.0, metrics.DrawdownHistory[0].Drawdown)
	assert.Equal(t, 255.0, metrics.DrawdownHistory[3].Drawdown) // Peak was 10495, current 10240
}

func TestCalculateDrawdownHistory(t *testing.T) {
	trader := &Trader{}
	
	balanceHistory := []storage.BalancePoint{
		{Time: time.Now(), Balance: 10000},
		{Time: time.Now().Add(time.Hour), Balance: 10500},     // New peak
		{Time: time.Now().Add(2 * time.Hour), Balance: 10300}, // Drawdown of 200
		{Time: time.Now().Add(3 * time.Hour), Balance: 10200}, // Drawdown of 300
		{Time: time.Now().Add(4 * time.Hour), Balance: 10600}, // New peak
		{Time: time.Now().Add(5 * time.Hour), Balance: 10400}, // Drawdown of 200
	}
	
	drawdownHistory := trader.calculateDrawdownHistory(balanceHistory)
	
	assert.Len(t, drawdownHistory, 6)
	
	// Test specific drawdown calculations
	assert.Equal(t, 0.0, drawdownHistory[0].Drawdown)
	assert.Equal(t, 0.0, drawdownHistory[0].DrawdownPercent)
	
	assert.Equal(t, 0.0, drawdownHistory[1].Drawdown) // New peak
	assert.Equal(t, 0.0, drawdownHistory[1].DrawdownPercent)
	
	assert.Equal(t, 200.0, drawdownHistory[2].Drawdown) // 10500 - 10300
	assert.InDelta(t, 1.90, drawdownHistory[2].DrawdownPercent, 0.01) // 200/10500 * 100
	
	assert.Equal(t, 300.0, drawdownHistory[3].Drawdown) // 10500 - 10200
	assert.InDelta(t, 2.86, drawdownHistory[3].DrawdownPercent, 0.01) // 300/10500 * 100
	
	assert.Equal(t, 0.0, drawdownHistory[4].Drawdown) // New peak
	assert.Equal(t, 0.0, drawdownHistory[4].DrawdownPercent)
	
	assert.Equal(t, 200.0, drawdownHistory[5].Drawdown) // 10600 - 10400
	assert.InDelta(t, 1.89, drawdownHistory[5].DrawdownPercent, 0.01) // 200/10600 * 100
}

func TestCalculatePerformanceMetrics_NoTrades(t *testing.T) {
	trader := newTestTrader()
	
	// Mock broker with just balance history
	trader.broker = &mockBroker{
		balanceHistory: []broker.BalancePoint{
			{Time: time.Now(), Balance: 10000},
		},
		maxDrawdown: 0.0,
	}
	
	trades := []*broker.Trade{}
	finalBalance := 10000.0
	percentChange := 0.0
	
	metrics := trader.calculatePerformanceMetrics(trades, finalBalance, percentChange)
	
	assert.Equal(t, 0, metrics.TotalTrades)
	assert.Equal(t, 0, metrics.WinningTrades)
	assert.Equal(t, 0, metrics.LosingTrades)
	assert.Equal(t, 0.0, metrics.TotalProfit)
	assert.Equal(t, 0.0, metrics.WinRate)
	assert.Equal(t, 0.0, metrics.AverageWin)
	assert.Equal(t, 0.0, metrics.AverageLoss)
	assert.Equal(t, 0.0, metrics.ProfitFactor)
	assert.Equal(t, 0.0, metrics.AverageMFE)
	assert.Equal(t, 0.0, metrics.AverageMAE)
	assert.Equal(t, 10000.0, metrics.FinalBalance)
	assert.Equal(t, 0.0, metrics.ReturnPercentage)
}

