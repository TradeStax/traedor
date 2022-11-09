package strategy

import (
	"github.com/tradestax/traedor/config"
	"github.com/tradestax/traedor/datafeed"

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
	config        config.StrategyConfig
	dataCache     []datafeed.Data
	indicatorChan chan Indicator
}

func NewMacdStrategy(c config.StrategyConfig, ic chan Indicator) IStrategy {
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

func (s *MacdStrategy) GetIndicatorFeed() chan Indicator {
	return s.indicatorChan
}

func (s *MacdStrategy) determineIndicator() {
	var ind Indicator
	if s.dataCache[0].Volume == 0 {
		ind.Direction = None
		s.indicatorChan <- ind
		return
	}
	macdVals := macd(s.dataCache)
	latestMacd := macdVals[len(macdVals)-1]
	if latestMacd > macdOverboughtVal || latestMacd < macdOversoldVal {
		// overbought or oversold, close
		ind.Direction = Close
		ind.Price = s.dataCache[9].Close
	} else if macdIncreasing(macdVals) {
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
