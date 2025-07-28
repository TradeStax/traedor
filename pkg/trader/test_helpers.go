package trader

import (
	"github.com/tradestax/traedor/internal/config"
	"github.com/tradestax/traedor/pkg/auth"
	"github.com/tradestax/traedor/pkg/broker"
	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/strategy/types"
)

// Shared test helpers for trader package tests

// mockBroker implements the broker.IBroker interface for testing
type mockBroker struct {
	trades          []*broker.Trade
	balanceHistory  []broker.BalancePoint
	maxDrawdown     float64
}

func (m *mockBroker) GetBalanceHistory() []broker.BalancePoint {
	return m.balanceHistory
}

func (m *mockBroker) GetMaxDrawdown() float64 {
	return m.maxDrawdown
}

func (m *mockBroker) AddData(data datafeed.Data) {}
func (m *mockBroker) SendTrade(trade broker.Trade) error { return nil }
func (m *mockBroker) Summary() {}
func (m *mockBroker) GetAccountStats() (*broker.Account, error) { return nil, nil }
func (m *mockBroker) GetTrades() ([]*broker.Trade, error) { return m.trades, nil }

// newTestTrader creates a trader with minimal config for testing
func newTestTrader() *Trader {
	cfg := &config.Config{
		AuthConfig: auth.Config{AuthHelper: "None"},
		Broker: broker.Config{
			Type:            "Futures",
			StartingBalance: 10000,
			BlackoutTimes: broker.BlackoutTimesStrings{
				StartTime: "17:00",
				EndTime:   "18:00",
				TimeZone:  "America/New_York",
			},
			Symbol: broker.Symbol{
				Name:       "ES",
				PointPrice: 50,
			},
		},
		Datafeeds: []datafeed.Config{},
		Strategy: []types.Config{
			{
				Type:   "SMA",
				Symbol: "ES",
				Params: types.Params{},
			},
		},
	}
	
	return NewTrader(cfg)
}

// newTestTraderWithBalance creates a trader with custom starting balance
func newTestTraderWithBalance(balance float64) *Trader {
	cfg := &config.Config{
		AuthConfig: auth.Config{AuthHelper: "None"},
		Broker: broker.Config{
			Type:            "Futures",
			StartingBalance: balance,
			BlackoutTimes: broker.BlackoutTimesStrings{
				StartTime: "17:00",
				EndTime:   "18:00",
				TimeZone:  "America/New_York",
			},
			Symbol: broker.Symbol{
				Name:       "ES",
				PointPrice: 50,
			},
		},
		Datafeeds: []datafeed.Config{},
		Strategy: []types.Config{
			{
				Type:   "SMA",
				Symbol: "ES",
				Params: types.Params{},
			},
		},
	}
	
	return NewTrader(cfg)
}