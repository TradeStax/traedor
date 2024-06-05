package strategy

import (
	"time"

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
	dataCache     []float64
	indicatorChan chan types.Indicator
	prevCloseTime time.Time
	currIndex     int
}

func NewRsiStrategy(c *types.Config, ic chan types.Indicator) types.IStrategy {
	return &RsiStrategy{
		config:        c,
		dataCache:     make([]float64, 10),
		indicatorChan: ic,
		prevCloseTime: time.Unix(0, 0),
		currIndex:     -1,
	}
}

func (s *RsiStrategy) AddData(data datafeed.Data) error {
	currCloseTime := time.Unix(data.Date, 0)
	if s.prevCloseTime.Year() > 2000 {
		currM := currCloseTime.Minute()
		prevM := s.prevCloseTime.Minute()
		// aggregate 5 min close array
		if (currM%5 == 0) && (prevM%5 != 0) && (s.currIndex != currM) {
			s.currIndex = currM
			for i := 0; i < 9; i++ {
				s.dataCache[i] = s.dataCache[i+1]
			}
		}
	}
	s.prevCloseTime = currCloseTime
	s.dataCache[9] = data.Close
	s.determineIndicator()
	return nil
}

func (s *RsiStrategy) GetIndicatorFeed() chan types.Indicator {
	return s.indicatorChan
}

func (s *RsiStrategy) determineIndicator() {
	ind := types.Indicator{
		Price:     s.dataCache[9],
		Direction: types.None,
	}
	// make sure the cache is filled before making decisions on incomplete data
	if s.dataCache[0] == float64(0) {
		s.indicatorChan <- ind
		return
	}
	rsiVals := talib.Rsi(s.dataCache, 2)
	latestRsi := rsiVals[len(rsiVals)-1]
	rsiInc := rsiIncreasing(rsiVals)
	ind.Price = s.dataCache[9]
	if latestRsi > overboughtVal || (latestRsi > oversoldVal && !rsiInc) {
		ind.Direction = types.Sell
	} else if latestRsi < oversoldVal || (latestRsi < overboughtVal && rsiInc) {
		ind.Direction = types.Buy
	}
	s.indicatorChan <- ind
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
