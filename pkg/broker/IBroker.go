package broker

import "github.com/tradestax/traedor/pkg/datafeed"

type IBroker interface {
	GetAccountStats() (*Account, error)
	AddData(datafeed.Data)
	SendTrade(Trade) error
	Summary()
	GetTrades() ([]*Trade, error)
	GetBalanceHistory() []BalancePoint
	GetMaxDrawdown() float64
}
