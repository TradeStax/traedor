package broker

import "github.com/tradestax/traedor/pkg/datafeed"

type IBroker interface {
	GetAccountStats() (*Account, error)
	AddData(datafeed.Data)
	SendTrade(Trade) error
	Summary()
}
