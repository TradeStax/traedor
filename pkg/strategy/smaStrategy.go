package strategy

import (
	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/strategy/types"
)

type SmaStrategy struct {
	config        *types.Config
	dataCache     []datafeed.Data
	indicatorChan chan types.Indicator
}

func NewSmaStrategy(c *types.Config, ic chan types.Indicator) types.IStrategy {
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

func (s *SmaStrategy) GetIndicatorFeed() chan types.Indicator {
	return s.indicatorChan
}

func (s *SmaStrategy) determineIndicator() {
	var ind types.Indicator
	if s.dataCache[0].Volume == 0 {
		ind.Direction = types.None
		s.indicatorChan <- ind
		return
	}
	diff := s.dataCache[9].Close - sma(s.dataCache)
	if diff > -0.1 && diff < 0.1 {
		ind.Direction = types.Close
		ind.Price = s.dataCache[9].Close
	} else if diff >= 0.1 {
		ind.Direction = types.Buy
		ind.Price = s.dataCache[9].Close
	} else {
		ind.Direction = types.Sell
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
