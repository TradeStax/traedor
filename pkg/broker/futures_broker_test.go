package broker

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tradestax/traedor/pkg/datafeed"
)

func TestFuturesBroker_BalanceTracking(t *testing.T) {
	brokerConfig := &Config{
		Type:            "Futures",
		StartingBalance: 10000,
		TradeQuantity:   1,
		BlackoutTimes: BlackoutTimesStrings{
			StartTime: "17:00",
			EndTime:   "18:00",
			TimeZone:  "America/New_York",
		},
		Symbol: Symbol{
			Name:       "ES",
			Margin:     4000,
			PointPrice: 50,
		},
		FeePerSide: 2.50,
	}
	
	broker := NewBroker(brokerConfig).(*FuturesBroker)
	
	// Add some market data
	data1 := datafeed.Data{
		Symbol: "ES",
		Date:   time.Now().Unix() * 1000,
		Close:  4500.0,
	}
	broker.AddData(data1)
	
	// Check initial balance history
	history := broker.GetBalanceHistory()
	assert.Len(t, history, 1)
	assert.Equal(t, 10000.0, history[0].Balance)
	
	// Execute a buy trade
	buyTrade := Trade{
		Symbol:    "ES",
		Operation: Buy,
		Price:     4500.0,
		Time:      data1.Date,
	}
	err := broker.SendTrade(buyTrade)
	require.NoError(t, err)
	
	// Add data to update position
	data2 := datafeed.Data{
		Symbol: "ES",
		Date:   time.Now().Add(time.Hour).Unix() * 1000,
		Close:  4510.0,
	}
	broker.AddData(data2)
	
	// Check balance history has grown
	history = broker.GetBalanceHistory()
	assert.GreaterOrEqual(t, len(history), 1)
	
	// Close the position
	closeTrade := Trade{
		Symbol:    "ES",
		Operation: Close,
		Price:     4510.0,
		Time:      data2.Date,
	}
	err = broker.SendTrade(closeTrade)
	require.NoError(t, err)
	
	// Check final balance is higher (profit trade)
	finalHistory := broker.GetBalanceHistory()
	assert.Greater(t, finalHistory[len(finalHistory)-1].Balance, 10000.0)
}

func TestFuturesBroker_MFEMAETracking(t *testing.T) {
	brokerConfig := &Config{
		Type:            "Futures",
		StartingBalance: 10000,
		TradeQuantity:   1,
		BlackoutTimes: BlackoutTimesStrings{
			StartTime: "17:00",
			EndTime:   "18:00",
			TimeZone:  "America/New_York",
		},
		Symbol: Symbol{
			Name:       "ES",
			Margin:     4000,
			PointPrice: 50,
		},
		FeePerSide: 2.50,
	}
	
	broker := NewBroker(brokerConfig).(*FuturesBroker)
	
	// Add initial market data
	data1 := datafeed.Data{
		Symbol: "ES",
		Date:   time.Now().Unix() * 1000,
		Close:  4500.0,
	}
	broker.AddData(data1)
	
	// Execute a buy trade
	buyTrade := Trade{
		Symbol:    "ES",
		Operation: Buy,
		Price:     4500.0,
		Time:      data1.Date,
	}
	err := broker.SendTrade(buyTrade)
	require.NoError(t, err)
	
	// Add data that moves in favor (higher price)
	data2 := datafeed.Data{
		Symbol: "ES",
		Date:   time.Now().Add(time.Minute).Unix() * 1000,
		Close:  4520.0, // +20 points favorable
	}
	broker.AddData(data2)
	
	// Add data that moves against (lower than favorable but still profitable)
	data3 := datafeed.Data{
		Symbol: "ES",
		Date:   time.Now().Add(2 * time.Minute).Unix() * 1000,
		Close:  4490.0, // -10 points adverse from entry
	}
	broker.AddData(data3)
	
	// Add data that moves back up
	data4 := datafeed.Data{
		Symbol: "ES",
		Date:   time.Now().Add(3 * time.Minute).Unix() * 1000,
		Close:  4510.0, // +10 points from entry
	}
	broker.AddData(data4)
	
	// Close the position
	closeTrade := Trade{
		Symbol:    "ES",
		Operation: Close,
		Price:     4510.0,
		Time:      data4.Date,
	}
	err = broker.SendTrade(closeTrade)
	require.NoError(t, err)
	
	// Get trades and check MFE/MAE
	trades, err := broker.GetTrades()
	require.NoError(t, err)
	assert.Len(t, trades, 1)
	
	trade := trades[0]
	assert.Equal(t, 4500.0, trade.OpenPrice)
	assert.Equal(t, 4510.0, trade.ClosePrice)
	
	// MFE should be 20 points (4520 - 4500) * 50 = $1000
	assert.Equal(t, 1000.0, trade.MFE)
	
	// MAE should be 10 points (4500 - 4490) * 50 = $500
	assert.Equal(t, 500.0, trade.MAE)
	
	// Check percentages
	assert.InDelta(t, 0.44, trade.MFEPercent, 0.01) // 20 points / 4500 * 100 ≈ 0.44%
	assert.InDelta(t, 0.22, trade.MAEPercent, 0.01) // 10 points / 4500 * 100 ≈ 0.22%
}

func TestFuturesBroker_DrawdownCalculation(t *testing.T) {
	brokerConfig := &Config{
		Type:            "Futures",
		StartingBalance: 10000,
		TradeQuantity:   1,
		BlackoutTimes: BlackoutTimesStrings{
			StartTime: "17:00",
			EndTime:   "18:00",
			TimeZone:  "America/New_York",
		},
		Symbol: Symbol{
			Name:       "ES",
			Margin:     4000,
			PointPrice: 50,
		},
		FeePerSide: 2.50,
	}
	
	broker := NewBroker(brokerConfig).(*FuturesBroker)
	
	// Simulate a series of trades that create drawdown
	// Start with profitable trade
	data1 := datafeed.Data{Symbol: "ES", Date: time.Now().Unix() * 1000, Close: 4500.0}
	broker.AddData(data1)
	
	buyTrade1 := Trade{Symbol: "ES", Operation: Buy, Price: 4500.0, Time: data1.Date}
	broker.SendTrade(buyTrade1)
	
	data2 := datafeed.Data{Symbol: "ES", Date: time.Now().Add(time.Hour).Unix() * 1000, Close: 4510.0}
	broker.AddData(data2)
	
	closeTrade1 := Trade{Symbol: "ES", Operation: Close, Price: 4510.0, Time: data2.Date}
	broker.SendTrade(closeTrade1)
	
	// Now lose money to create drawdown
	data3 := datafeed.Data{Symbol: "ES", Date: time.Now().Add(2 * time.Hour).Unix() * 1000, Close: 4510.0}
	broker.AddData(data3)
	
	buyTrade2 := Trade{Symbol: "ES", Operation: Buy, Price: 4510.0, Time: data3.Date}
	broker.SendTrade(buyTrade2)
	
	data4 := datafeed.Data{Symbol: "ES", Date: time.Now().Add(3 * time.Hour).Unix() * 1000, Close: 4490.0}
	broker.AddData(data4)
	
	closeTrade2 := Trade{Symbol: "ES", Operation: Close, Price: 4490.0, Time: data4.Date}
	broker.SendTrade(closeTrade2)
	
	// Check max drawdown
	maxDrawdown := broker.GetMaxDrawdown()
	assert.Greater(t, maxDrawdown, 0.0)
	
	// Get balance history and verify it tracks properly
	history := broker.GetBalanceHistory()
	assert.GreaterOrEqual(t, len(history), 3)
	
	// Find peak balance and verify drawdown calculation
	var peak float64
	for _, bp := range history {
		if bp.Balance > peak {
			peak = bp.Balance
		}
	}
	
	// Find maximum drawdown from peak
	var maxDD float64
	for _, bp := range history {
		if peak > bp.Balance {
			dd := peak - bp.Balance
			if dd > maxDD {
				maxDD = dd
			}
		}
	}
	
	assert.Equal(t, maxDD, maxDrawdown)
}

func TestFuturesBroker_GetAccountStats(t *testing.T) {
	brokerConfig := &Config{
		Type:            "Futures",
		StartingBalance: 10000,
		TradeQuantity:   1,
		BlackoutTimes: BlackoutTimesStrings{
			StartTime: "17:00",
			EndTime:   "18:00",
			TimeZone:  "America/New_York",
		},
		Symbol: Symbol{
			Name:       "ES",
			Margin:     4000,
			PointPrice: 50,
		},
		FeePerSide: 2.50,
	}
	
	broker := NewBroker(brokerConfig).(*FuturesBroker)
	
	// Check initial account stats
	account, err := broker.GetAccountStats()
	require.NoError(t, err)
	assert.Equal(t, 10000.0, account.Balance())
	
	// Execute some trades and check account updates
	data := datafeed.Data{Symbol: "ES", Date: time.Now().Unix() * 1000, Close: 4500.0}
	broker.AddData(data)
	
	buyTrade := Trade{Symbol: "ES", Operation: Buy, Price: 4500.0, Time: data.Date}
	broker.SendTrade(buyTrade)
	
	// Account should show margin usage
	account, err = broker.GetAccountStats()
	require.NoError(t, err)
	assert.Less(t, account.Balance(), 10000.0) // Should be reduced by fees
}

func TestFuturesBroker_MultiplePositions(t *testing.T) {
	brokerConfig := &Config{
		Type:            "Futures",
		StartingBalance: 50000, // Higher balance for multiple positions
		TradeQuantity:   1,
		BlackoutTimes: BlackoutTimesStrings{
			StartTime: "17:00",
			EndTime:   "18:00",
			TimeZone:  "America/New_York",
		},
		Symbol: Symbol{
			Name:       "ES",
			Margin:     4000,
			PointPrice: 50,
		},
		FeePerSide: 2.50,
	}
	
	broker := NewBroker(brokerConfig).(*FuturesBroker)
	
	// Test multiple consecutive positions
	for i := 0; i < 3; i++ {
		data := datafeed.Data{
			Symbol: "ES",
			Date:   time.Now().Add(time.Duration(i) * time.Hour).Unix() * 1000,
			Close:  4500.0 + float64(i*5), // Varying prices
		}
		broker.AddData(data)
		
		buyTrade := Trade{
			Symbol:    "ES",
			Operation: Buy,
			Price:     data.Close,
			Time:      data.Date,
		}
		broker.SendTrade(buyTrade)
		
		// Update with different price
		dataClose := datafeed.Data{
			Symbol: "ES",
			Date:   data.Date + 30*60*1000, // 30 minutes later
			Close:  data.Close + 10, // Always profitable
		}
		broker.AddData(dataClose)
		
		closeTrade := Trade{
			Symbol:    "ES",
			Operation: Close,
			Price:     dataClose.Close,
			Time:      dataClose.Date,
		}
		broker.SendTrade(closeTrade)
	}
	
	// Check that all trades were recorded
	trades, err := broker.GetTrades()
	require.NoError(t, err)
	assert.Len(t, trades, 3)
	
	// Check balance increased
	account, _ := broker.GetAccountStats()
	assert.Greater(t, account.Balance(), 50000.0)
}