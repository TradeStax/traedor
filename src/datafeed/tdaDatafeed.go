package datafeed

import (
	"time"

	"github.com/tradestax/traedor/config"
)

func NewTDADatafeed(c config.Config) *Datafeed {
	duration, err := time.ParseDuration(c.Interval)
	if err != nil {
		panic(err)
	}
	df := &Datafeed{
		config:    c,
		dataChan:  make(chan Data),
		errorChan: make(chan error),
	}
	go df.tdaDatafeed(duration)
	return df
}

func (d *Datafeed) tdaDatafeed(duration time.Duration) {
}
