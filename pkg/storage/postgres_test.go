package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tradestax/traedor/pkg/broker"
	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/strategy/types"
)

func TestPostgresStorage_SaveAndGetRun(t *testing.T) {
	// This would require a test database connection
	// For now, we'll write the test structure
	t.Skip("Requires database connection")
	
	storage := &PostgresStorage{}
	
	config := RunConfig{
		Symbol:    "ES",
		Timeframe: "5m",
		StartTime: time.Now().Add(-24 * time.Hour),
		EndTime:   time.Now(),
		Broker: BrokerConfig{
			StartingBalance: 10000,
		},
	}
	
	// Create run
	run, err := storage.CreateRun(config)
	require.NoError(t, err)
	assert.NotEmpty(t, run.ID)
	
	// Get run
	retrievedRun, err := storage.GetRun(run.ID)
	require.NoError(t, err)
	assert.Equal(t, run.ID, retrievedRun.ID)
	assert.Equal(t, config.Symbol, retrievedRun.Config.Symbol)
	assert.Equal(t, RunStatusPending, retrievedRun.Status)
}

func TestPostgresStorage_SaveTrade(t *testing.T) {
	t.Skip("Requires database connection")
	
	storage := &PostgresStorage{}
	runID := "test-run-id"
	
	trade := &broker.Trade{
		Symbol:     "ES",
		Operation:  broker.Buy,
		Quantity:   1,
		OpenPrice:  4500.0,
		ClosePrice: 4510.0,
		OpenTime:   time.Now().Unix(),
		CloseTime:  time.Now().Add(time.Hour).Unix(),
		Net:        100.0,
		MFE:        150.0,
		MAE:        -50.0,
		MFEPercent: 3.33,
		MAEPercent: -1.11,
	}
	
	err := storage.SaveTrade(runID, trade)
	require.NoError(t, err)
	
	// Get trades
	trades, err := storage.GetTrades(runID)
	require.NoError(t, err)
	assert.Len(t, trades, 1)
	assert.Equal(t, trade.Symbol, trades[0].Symbol)
	assert.Equal(t, trade.MFE, trades[0].MFE)
	assert.Equal(t, trade.MAE, trades[0].MAE)
}

func TestPostgresStorage_SavePerformanceMetrics(t *testing.T) {
	t.Skip("Requires database connection")
	
	storage := &PostgresStorage{}
	runID := "test-run-id"
	
	metrics := &PerformanceMetrics{
		TotalTrades:       10,
		WinningTrades:     6,
		LosingTrades:      4,
		TotalProfit:       1000.0,
		MaxDrawdown:       500.0,
		MaxDrawdownPercent: 5.0,
		WinRate:           60.0,
		AverageWin:        200.0,
		AverageLoss:       -100.0,
		ProfitFactor:      2.0,
		FinalBalance:      11000.0,
		ReturnPercentage:  10.0,
		AverageMFE:        150.0,
		AverageMFEPercent: 1.5,
		AverageMAE:        -75.0,
		AverageMAEPercent: -0.75,
		BalanceHistory: []BalancePoint{
			{Time: time.Now(), Balance: 10000},
			{Time: time.Now().Add(time.Hour), Balance: 10500},
			{Time: time.Now().Add(2 * time.Hour), Balance: 11000},
		},
		DrawdownHistory: []DrawdownPoint{
			{Time: time.Now(), Drawdown: 0, DrawdownPercent: 0},
			{Time: time.Now().Add(time.Hour), Drawdown: 200, DrawdownPercent: 2},
			{Time: time.Now().Add(2 * time.Hour), Drawdown: 0, DrawdownPercent: 0},
		},
	}
	
	err := storage.UpdateRunStatus(runID, RunStatusCompleted, metrics)
	require.NoError(t, err)
	
	// Get run with metrics
	run, err := storage.GetRun(runID)
	require.NoError(t, err)
	assert.Equal(t, RunStatusCompleted, run.Status)
	assert.NotNil(t, run.PerformanceMetrics)
	assert.Equal(t, metrics.TotalTrades, run.PerformanceMetrics.TotalTrades)
	assert.Equal(t, metrics.AverageMFE, run.PerformanceMetrics.AverageMFE)
	assert.Len(t, run.PerformanceMetrics.BalanceHistory, 3)
}

func TestPostgresStorage_SaveTickData(t *testing.T) {
	t.Skip("Requires database connection")
	
	storage := &PostgresStorage{}
	
	ticks := []datafeed.Data{
		{
			Symbol: "ES",
			Date:   time.Now().Unix() * 1000,
			Open:   4500.0,
			High:   4510.0,
			Low:    4495.0,
			Close:  4505.0,
			Volume: 1000,
		},
		{
			Symbol: "ES",
			Date:   time.Now().Add(time.Minute).Unix() * 1000,
			Open:   4505.0,
			High:   4515.0,
			Low:    4500.0,
			Close:  4510.0,
			Volume: 1500,
		},
	}
	
	err := storage.SaveTickData(ticks)
	require.NoError(t, err)
}

func TestPostgresStorage_SaveSignal(t *testing.T) {
	t.Skip("Requires database connection")
	
	storage := &PostgresStorage{}
	runID := "test-run-id"
	
	signal := Signal{
		RunID:     runID,
		Time:      time.Now(),
		Symbol:    "ES",
		Direction: types.Indicator{Direction: types.Buy},
		Price:     4500.0,
	}
	
	err := storage.SaveSignal(runID, signal)
	require.NoError(t, err)
	
	// Get signals
	signals, err := storage.GetSignals(runID)
	require.NoError(t, err)
	assert.Len(t, signals, 1)
	assert.Equal(t, signal.Symbol, signals[0].Symbol)
	assert.Equal(t, signal.Direction, signals[0].Direction)
}