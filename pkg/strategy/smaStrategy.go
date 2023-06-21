package strategy

import (
	"github.com/tradestax/traedor/pkg/datafeed"
)

type SmaStrategy struct {
	config        *Config
	dataCache     []datafeed.Data
	indicatorChan chan Indicator
}

func NewSmaStrategy(c *Config, ic chan Indicator) IStrategy {
	return &SmaStrategy{
		config:        c,
		dataCache:     make([]datafeed.Data, 10),
		indicatorChan: ic,
	}
}

func (s *SmaStrategy) AddData(data datafeed.Data) error {
	for i := 0; i < len(s.dataCache)-1; i++ {
		s.dataCache[i] = s.dataCache[i+1]
	}
	s.dataCache[9] = data
	go s.determineIndicator()
	return nil
}

func (s *SmaStrategy) GetIndicatorFeed() chan Indicator {
	return s.indicatorChan
}

func (s *SmaStrategy) determineIndicator() {
	var ind Indicator
	if s.dataCache[0].Volume == 0 {
		ind.Direction = None
		s.indicatorChan <- ind
		return
	}
	diff := s.dataCache[9].Close - sma(s.dataCache)
	if diff > -0.1 && diff < 0.1 {
		ind.Direction = Close
		ind.Price = s.dataCache[9].Close
	} else if diff >= 0.1 {
		ind.Direction = Buy
		ind.Price = s.dataCache[9].Close
	} else {
		ind.Direction = Sell
		ind.Price = s.dataCache[9].Close
	}
	s.indicatorChan <- ind
}

func sma(data []datafeed.Data) float64 {
	sum := float64(0)
	for _, d := range data {
		sum += d.Close
	}
	return sum / float64(len(data))
}
