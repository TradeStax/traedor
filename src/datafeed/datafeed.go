package datafeed

import "time"

type Datafeed struct {
	config   Config
	dataChan chan Data
}

type Config struct {
	Symbol   string
	Interval string
}

func NewDatafeed(c Config) *Datafeed {
	df := &Datafeed{
		config:   c,
		dataChan: make(chan Data),
	}
	go func() {
		for {
			df.dataChan <- Data{
				high:   1.0,
				low:    1.0,
				open:   1.0,
				close:  1.0,
				volume: 1,
			}
			time.Sleep(5 * time.Second)
		}
	}()
	return df
}

func (d *Datafeed) GetDatafeed() chan Data {
	return d.dataChan
}
