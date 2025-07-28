package datafeed

import (
	"context"
	"fmt"
	"time"

	"github.com/tradestax/traedor/pkg/auth"
	"github.com/tradestax/traedor/pkg/storage"
)

type Config struct {
	DataPath  string
	Fields    string
	Interval  string
	Service   string
	Symbol    string
	Type      string
	StartTime int64
	EndTime   int64
	Print     bool
	Storage   interface{} // For database datafeed
}

func NewDatafeed(c *Config, ah auth.IAuthHelper, dc chan Data, ec chan error) IDatafeed {
	var df IDatafeed
	switch c.Type {
	case "Generated":
		df = NewLocalDatafeed(c, dc, ec)
	case "CSV":
		df = NewLocalDatafeed(c, dc, ec)
	case "TDA":
		df = NewTDADatafeed(c, ah, dc, ec)
	case "Database":
		if store, ok := c.Storage.(storage.IStorage); ok {
			df = NewDBDatafeed(c, store, dc, ec)
		} else {
			panic(fmt.Errorf("Database datafeed requires storage instance"))
		}
	default:
		panic(fmt.Errorf("Invalid datafeed specified"))
	}
	return df
}

func NewLocalDatafeed(c *Config, dc chan Data, ec chan error) *Datafeed {
	duration, err := time.ParseDuration(c.Interval)
	if err != nil {
		panic(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Datafeed{
		config:    c,
		dataChan:  dc,
		errorChan: ec,
		duration:  duration,
		startTime: c.StartTime,
		endTime:   c.EndTime,
		ctx:       ctx,
		cancel:    cancel,
	}
}
