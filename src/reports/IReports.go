package reports

import "github.com/tradestax/traedor/datafeed"

type IReports interface {
	AddData(datafeed.Data)
	CreateReports()
}
