package datafeed

import (
	"fmt"

	"github.com/tradestax/traedor/config"
)

func NewDatafeed(c config.Config) IDatafeed {
	var df IDatafeed
	switch c.Datafeed {
	case "Generated":
		df = NewGeneratedDatafeed(c)
	case "CSV":
		df = NewCSVDatafeed(c)
	default:
		panic(fmt.Errorf("Invalid datafeed specified"))
	}
	return df
}
