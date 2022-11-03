package strategy

import (
	"github.com/tradestax/traedor/config"
	"github.com/tradestax/traedor/datafeed"

	"github.com/markcheno/go-talib"
)

const (
	overboughtVal = 70.0
	oversoldVal   = 30.0
)

type RsiStrategy struct {
	config        config.StrategyConfig
	dataCache     []datafeed.Data
	indicatorChan chan Indicator
}

func NewRsiStrategy(c config.StrategyConfig, ic chan Indicator) IStrategy {
	return &RsiStrategy{
		config:        c,
		dataCache:     make([]datafeed.Data, 10),
		indicatorChan: ic,
	}
}

func (s *RsiStrategy) AddData(data datafeed.Data) error {
	for i := 0; i < len(s.dataCache)-1; i++ {
		s.dataCache[i] = s.dataCache[i+1]
	}
	s.dataCache[9] = data
	go s.determineIndicator()
	return nil
}

func (s *RsiStrategy) GetIndicatorFeed() chan Indicator {
	return s.indicatorChan
}

func (s *RsiStrategy) determineIndicator() {
	var ind Indicator
	if s.dataCache[0].Volume == 0 {
		ind.Direction = None
		s.indicatorChan <- ind
		return
	}
	rsiVals := rsi(s.dataCache)
	latestRsi := rsiVals[len(rsiVals)-1]
	if latestRsi > overboughtVal || latestRsi < oversoldVal {
		// overbought or oversold, close
		ind.Direction = Close
		ind.Price = s.dataCache[9].Close
	} else if rsiIncreasing(rsiVals) {
		// increasing, buy
		ind.Direction = Buy
		ind.Price = s.dataCache[9].Close
	} else {
		// decreasing, sell
		ind.Direction = Sell
		ind.Price = s.dataCache[9].Close
	}
	s.indicatorChan <- ind
}

func rsi(data []datafeed.Data) []float64 {
	closePrice := make([]float64, len(data))
	for i, d := range data {
		closePrice[i] = d.Close
	}
	return talib.Rsi(closePrice, 2)
}

func rsiIncreasing(rsiVals []float64) bool {
	trendConf := 0
	// loops through RSI from back to front
	for i := len(rsiVals) - 1; i > 0; i-- {
		if rsiVals[i] > rsiVals[i-1] {
			trendConf++
			if trendConf > 1 {
				return true
			}
		} else {
			return false
		}
	}
	return false
}
