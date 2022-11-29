package strategy

import "github.com/tradestax/traedor/pkg/datafeed"

type IStrategy interface {
	AddData(datafeed.Data) error
	GetIndicatorFeed() chan Indicator
}
