package strategy

import (
	"github.com/tradestax/traedor/config"
	"github.com/tradestax/traedor/datafeed"
)

type EnsembleStrategy struct {
	config        []config.StrategyConfig
	dataCache     []datafeed.Data
	indicatorChan chan Indicator
}

func NewEnsembleStrategy(c []config.StrategyConfig, ic chan Indicator) *EnsembleStrategy {
	return &EnsembleStrategy{
		config:        c,
		dataCache:     make([]datafeed.Data, 10),
		indicatorChan: ic,
	}
}

func (s *EnsembleStrategy) AddData(data datafeed.Data) error {
	for i := 0; i < len(s.dataCache)-1; i++ {
		s.dataCache[i] = s.dataCache[i+1]
	}
	s.dataCache[9] = data
	return nil
}

func (s *EnsembleStrategy) GetIndicatorFeed() chan Indicator {
	return s.indicatorChan
}
