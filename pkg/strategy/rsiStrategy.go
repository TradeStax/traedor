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
	lastRSI       float64
	lastSignal    int
}

func NewRsiStrategy(c *types.Config, ic chan types.Indicator) types.IStrategy {
	return &RsiStrategy{
		config:        c,
		dataCache:     make([]datafeed.Data, 10),
		indicatorChan: ic,
		lastRSI:       50.0, // neutral starting value
		lastSignal:    types.None,
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
	ind.Direction = types.None // Default to no signal
	ind.Price = s.dataCache[9].Close
	
	if s.dataCache[0].Volume == 0 {
		s.indicatorChan <- ind
		return
	}
	
	rsiVals := rsi(s.dataCache)
	if len(rsiVals) == 0 {
		s.indicatorChan <- ind
		return
	}
	
	latestRsi := rsiVals[len(rsiVals)-1]
	
	// Only signal on RSI crossovers, not every tick
	if s.lastRSI > oversoldVal && latestRsi <= oversoldVal {
		// RSI just entered oversold territory - potential buy signal
		ind.Direction = types.Buy
		s.lastSignal = types.Buy
	} else if s.lastRSI < overboughtVal && latestRsi >= overboughtVal {
		// RSI just entered overbought territory - potential sell signal
		ind.Direction = types.Sell
		s.lastSignal = types.Sell
	}
	// Note: Removed the automatic "Close" signal to prevent overtrading
	
	s.lastRSI = latestRsi
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
