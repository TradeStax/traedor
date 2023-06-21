package datafeed

import (
	"fmt"
	"math"
	"time"
)

type Datafeed struct {
	config    *Config
	dataChan  chan Data
	errorChan chan error
	duration  time.Duration
	startTime int64
	endTime   int64
}

const testDataMax = 100

func (d *Datafeed) Start() {
	switch d.config.Type {
	case "Generated":
		switch d.config.Symbol {
		case "ones":
			go d.onesData(d.duration)
		case "sin":
			go d.sinData(d.duration)
		default:
			panic(fmt.Errorf("Unrecognized symbol provided"))
		}
	case "CSV":
		go d.csvDatafeed(d.duration)
	case "SC":
		go d.scDatafeed(d.duration)
	default:
		panic(fmt.Errorf("Invalid datafeed specified"))
	}
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
