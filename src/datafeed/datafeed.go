package datafeed

import (
	"fmt"
	"math"
	"time"
)

type Datafeed struct {
	config    Config
	dataChan  chan Data
	errorChan chan error
}

type Config struct {
	Symbol   string
	Interval string
}

const testDataMax = 100

func NewDatafeed(c Config) *Datafeed {
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
			high:   1.0,
			low:    1.0,
			open:   1.0,
			close:  1.0,
			volume: 1,
		}
		time.Sleep(duration)
	}
	d.errorChan <- fmt.Errorf("Test Completed")
}

func (d *Datafeed) sinData(duration time.Duration) {
	for i := 0; i < testDataMax; i++ {
		v := math.Sin(float64(i)/float64(testDataMax/16)) + 1
		d.dataChan <- Data{
			high:   v,
			low:    v,
			open:   v,
			close:  v,
			volume: 1,
		}
		time.Sleep(duration)
	}
	d.errorChan <- fmt.Errorf("Test Completed")
}
