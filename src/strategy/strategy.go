package strategy

import "github.com/tradestax/traedor/datafeed"

type Strategy struct {
	dataCache     []datafeed.Data
	indicatorChan chan Indicator
}

func NewStrategy() *Strategy {
	return &Strategy{
		dataCache:     make([]datafeed.Data, 10),
		indicatorChan: make(chan Indicator),
	}
}

func (s *Strategy) AddData(data datafeed.Data) error {
	for i := 0; i < len(s.dataCache)-1; i++ {
		s.dataCache[i] = s.dataCache[i+1]
	}
	s.dataCache[9] = data
	go s.determineIndicator()
	return nil
}

func (s *Strategy) GetIndicatorFeed() chan Indicator {
	return s.indicatorChan
}

func (s *Strategy) determineIndicator() {
	var ind Indicator
	if s.dataCache[0].Volume == 0 {
		ind.Direction = None
		s.indicatorChan <- ind
		return
	}
	diff := s.dataCache[9].Close - sma(s.dataCache)
	if diff > -0.1 && diff < 0.1 {
		ind.Direction = Close
	} else if diff >= 0.1 {
		ind.Direction = Buy
	} else {
		ind.Direction = Sell
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
