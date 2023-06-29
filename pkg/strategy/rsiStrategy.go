package strategy

import (
	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/strategy/types"

	"github.com/markcheno/go-talib"
)

const (
	overboughtVal = 70.0
	oversoldVal   = 30.0
)

type RsiStrategy struct {
	config        *types.Config
	dataCache     []datafeed.Data
	indicatorChan chan types.Indicator
}

func NewRsiStrategy(c *types.Config, ic chan types.Indicator) types.IStrategy {
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
	s.determineIndicator()
	return nil
}

func (s *RsiStrategy) GetIndicatorFeed() chan types.Indicator {
	return s.indicatorChan
}

func (s *RsiStrategy) determineIndicator() {
	var ind types.Indicator
	if s.dataCache[0].Volume == 0 {
		ind.Direction = types.None
		s.indicatorChan <- ind
		return
	}
	rsiVals := rsi(s.dataCache)
	latestRsi := rsiVals[len(rsiVals)-1]
	if latestRsi > overboughtVal || latestRsi < oversoldVal {
		// overbought or oversold, close
		ind.Direction = types.Close
		ind.Price = s.dataCache[9].Close
	} else if rsiIncreasing(rsiVals) {
		// increasing, buy
		ind.Direction = types.Buy
		ind.Price = s.dataCache[9].Close
	} else {
		// decreasing, sell
		ind.Direction = types.Sell
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
