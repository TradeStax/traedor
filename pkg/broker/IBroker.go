package broker

import "github.com/tradestax/traedor/pkg/types"

type IBroker interface {
	GetAccountStats() (*Account, error)
	AddData(types.Data)
	SendTrade(Trade) error
	Summary()
	GetTrades() ([]*Trade, error)
	GetBalanceHistory() []BalancePoint
	GetMaxDrawdown() float64
}
