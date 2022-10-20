package datafeed

import (
	"fmt"
	"math"
	"time"

	"github.com/tradestax/traedor/config"
)

type Datafeed struct {
	config    config.Config
	dataChan  chan Data
	errorChan chan error
}

const testDataMax = 100

func NewGeneratedDatafeed(c config.Config) *Datafeed {
	duration, err := time.ParseDuration(c.Interval)
	if err != nil {
		panic(err)
	}
	df := &Datafeed{
		config:    c,
		dataChan:  make(chan Data),
		errorChan: make(chan error),
	}
	switch c.Symbol {
	case "ones":
		go df.onesData(duration)
	case "sin":
		go df.sinData(duration)
	default:
		panic(fmt.Errorf("Unrecognized symbol provided"))
	}
	return df
}

func (d *Datafeed) GetDatafeed() chan Data {
	return d.dataChan
}

func (d *Datafeed) GetErrorChan() chan error {
	return d.errorChan
}

func (d *Datafeed) onesData(duration time.Duration) {
	for i := 0; i < testDataMax; i++ {
		d.dataChan <- Data{
			High:   1.0,
			Low:    1.0,
			Open:   1.0,
			Close:  1.0,
			Volume: 1.0,
		}
		time.Sleep(duration)
	}
	d.errorChan <- fmt.Errorf("Test Completed")
}

func (d *Datafeed) sinData(duration time.Duration) {
	for i := 0; i < testDataMax; i++ {
		v := math.Sin(float64(i)/float64(testDataMax/16)) + 1
		d.dataChan <- Data{
			High:   v,
			Low:    v,
			Open:   v,
			Close:  v,
			Volume: 1.0,
		}
		time.Sleep(duration)
	}
	d.errorChan <- fmt.Errorf("Test Completed")
}
