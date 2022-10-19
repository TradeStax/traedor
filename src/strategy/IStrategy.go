package strategy

import "github.com/tradestax/traedor/datafeed"

type IStrategy interface {
	AddData(datafeed.Data) error
	GetIndicatorFeed() chan Indicator
}
