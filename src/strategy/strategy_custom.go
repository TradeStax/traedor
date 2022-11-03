package strategy

import (
	"github.com/tradestax/traedor/config"
	"github.com/tradestax/traedor/datafeed"
)

var (
	customStrategies = map[string]stratBuilder{
		"custom": NewCustomStrategy,
	}
)

type CustomStrategy struct {
	config        config.StrategyConfig
	dataCache     []datafeed.Data
	indicatorChan chan Indicator
}

func NewCustomStrategy(c config.StrategyConfig, ic chan Indicator) IStrategy {
	return &CustomStrategy{
		config:        c,
		dataCache:     make([]datafeed.Data, 10),
		indicatorChan: ic,
	}
}

func (s *CustomStrategy) AddData(data datafeed.Data) error {
	for i := 0; i < len(s.dataCache)-1; i++ {
		s.dataCache[i] = s.dataCache[i+1]
	}
	s.dataCache[9] = data
	go s.determineIndicator()
	return nil
}

func (s *CustomStrategy) GetIndicatorFeed() chan Indicator {
	return s.indicatorChan
}

func (s *CustomStrategy) determineIndicator() {
	s.indicatorChan <- Indicator{
		Direction: None,
	}
}
