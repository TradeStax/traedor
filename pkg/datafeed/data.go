package datafeed

import "log"

type Data struct {
	Date   int64
	High   float64
	Low    float64
	Open   float64
	Close  float64
	Volume float64
	Symbol string
}

func (d *Data) Print() {
	log.Printf("Symbol: %v\nDate: %v\nHigh: %v\nLow: %v\nOpen: %v\nClose: %v\nVolume: %v\n",
		d.Symbol, d.Date, d.High, d.Low, d.Open, d.Close, d.Volume)
}
