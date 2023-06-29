package strategy

import (
	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/strategy/types"

	"github.com/markcheno/go-talib"
)

const (
	macdSlowPeriod    = 5
	macdFastPeriod    = 1
	macdSignalPeriod  = 2
	macdOverboughtVal = 0.5
	macdOversoldVal   = -0.5
)

type MacdStrategy struct {
	config        *types.Config
	dataCache     []datafeed.Data
	indicatorChan chan types.Indicator
}

func NewMacdStrategy(c *types.Config, ic chan types.Indicator) types.IStrategy {
	return &MacdStrategy{
		config:        c,
		dataCache:     make([]datafeed.Data, 10),
		indicatorChan: ic,
	}
}

func (s *MacdStrategy) AddData(data datafeed.Data) error {
	for i := 0; i < len(s.dataCache)-1; i++ {
		s.dataCache[i] = s.dataCache[i+1]
	}
	s.dataCache[9] = data
	s.determineIndicator()
	return nil
}

func (s *MacdStrategy) GetIndicatorFeed() chan types.Indicator {
	return s.indicatorChan
}

func (s *MacdStrategy) determineIndicator() {
	var ind types.Indicator
	if s.dataCache[0].Volume == 0 {
		ind.Direction = types.None
		s.indicatorChan <- ind
		return
	}
	macdVals := macd(s.dataCache)
	latestMacd := macdVals[len(macdVals)-1]
	if latestMacd > macdOverboughtVal || latestMacd < macdOversoldVal {
		// overbought or oversold, close
		ind.Direction = types.Close
		ind.Price = s.dataCache[9].Close
		ind.Time = s.dataCache[9].Date
	} else if macdIncreasing(macdVals) {
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

func macd(data []datafeed.Data) []float64 {
	closePrice := make([]float64, len(data))
	for i, d := range data {
		closePrice[i] = d.Close
	}
	_, signals, _ := talib.Macd(closePrice, macdFastPeriod, macdSlowPeriod, macdSignalPeriod)
	return signals
}

func macdIncreasing(macdVals []float64) bool {
	trendConf := 0
	// loops through MACD from back to front
	for i := len(macdVals) - 1; i > 0; i-- {
		if macdVals[i] > macdVals[i-1] {
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
