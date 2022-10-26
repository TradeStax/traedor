package datafeed

import (
	"fmt"

	"github.com/tradestax/traedor/auth"
	"github.com/tradestax/traedor/config"
)

func NewDatafeed(c *config.Config, ah auth.IAuthHelper) IDatafeed {
	var df IDatafeed
	switch c.Datafeed {
	case "Generated":
		df = NewGeneratedDatafeed(c)
	case "CSV":
		df = NewCSVDatafeed(c)
	case "TDA":
		df = NewTDADatafeed(c, ah)
	default:
		panic(fmt.Errorf("Invalid datafeed specified"))
	}
	return df
}
