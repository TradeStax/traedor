package datafeed

type Datafeed struct {
	config   Config
	dataChan chan Data
}

type Config struct {
	symbol   string
	interval string
}

func NewDatafeed(c Config) *Datafeed {
	return &Datafeed{
		config:   c,
		dataChan: make(chan Data),
	}
}

func (d *Datafeed) GetDataFeed() chan Data {
	return d.dataChan
}
