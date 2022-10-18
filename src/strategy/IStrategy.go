package strategy

import "github.com/TradeStax/traedor/datafeed"

type IStrategy interface {
	AddData(datafeed.Data) error
	GetIndicatorFeed() chan Indicator
}
