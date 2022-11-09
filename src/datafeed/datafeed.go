package datafeed

import (
	"fmt"
	"time"

	"github.com/tradestax/traedor/auth"
	"github.com/tradestax/traedor/config"
)

func NewDatafeed(c *config.DatafeedConfig, ah auth.IAuthHelper, dc chan Data, ec chan error) IDatafeed {
	var df IDatafeed
	switch c.Type {
	case "Generated":
		df = NewLocalDatafeed(c, dc, ec)
	case "CSV":
		df = NewLocalDatafeed(c, dc, ec)
	case "TDA":
		df = NewTDADatafeed(c, ah, dc, ec)
	default:
		panic(fmt.Errorf("Invalid datafeed specified"))
	}
	return df
}

func NewLocalDatafeed(c *config.DatafeedConfig, dc chan Data, ec chan error) *Datafeed {
	duration, err := time.ParseDuration(c.Interval)
	if err != nil {
		panic(err)
	}
	return &Datafeed{
		config:    c,
		dataChan:  dc,
		errorChan: ec,
		duration:  duration,
	}
}
