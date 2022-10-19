package strategy

import "github.com/tradestax/traedor/datafeed"

type Strategy struct {
	indicatorChan chan Indicator
}

func NewStrategy() *Strategy {
	return &Strategy{
		indicatorChan: make(chan Indicator),
	}
}

func (s *Strategy) AddData(datafeed.Data) error {
	go func() {
		s.indicatorChan <- Indicator{Direction: None}
	}()
	return nil
}

func (s *Strategy) GetIndicatorFeed() chan Indicator {
	return s.indicatorChan
}
