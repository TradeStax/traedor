package emacrossover

import (
	"time"

	"github.com/markcheno/go-talib"
	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/strategy/types"
)

const (
	fastEmaLength = 7
	slowEmaLength = 21
	cacheLength   = slowEmaLength * 2
	slowEmaGap    = 2
)

var (
	loc, _ = time.LoadLocation("America/New_York")
)

type cross struct {
	above bool
	below bool
}

type EmaCrossover struct {
	shortTermClose []float64
	longTermClose  []float64
	prevCloseTime  time.Time
	indicatorChan  chan types.Indicator
	lastSend       int
	shortIndex     int
	longIndex      int
}

func NewEmaCrossoverStrategy(c *types.Config, ic chan types.Indicator) types.IStrategy {
	sc := &EmaCrossover{
		shortTermClose: make([]float64, cacheLength),
		longTermClose:  make([]float64, cacheLength),
		prevCloseTime:  time.Unix(0, 0),
		indicatorChan:  ic,
		lastSend:       types.None,
		shortIndex:     -1,
		longIndex:      -1,
	}
	return sc
}

func (s *EmaCrossover) AddData(data datafeed.Data) error {
	currCloseTime := time.Unix(data.Date, 0)
	if s.prevCloseTime.Year() > 2000 {
		currH := currCloseTime.Hour()
		currM := currCloseTime.Minute()
		prevM := s.prevCloseTime.Minute()
		// aggregate 5 min close array
		if (currM%5 == 0) && (prevM%5 != 0) && (s.shortIndex != currM) {
			s.shortIndex = currM
			for i := 0; i < cacheLength-1; i++ {
				s.shortTermClose[i] = s.shortTermClose[i+1]
			}
		}
		// aggregate 1 hr close array
		if (currM%60 == 0) && (prevM%60 != 0) && (s.longIndex != currH) {
			s.longIndex = currH
			for i := 0; i < cacheLength-1; i++ {
				s.longTermClose[i] = s.longTermClose[i+1]
			}
		}
	}
	s.prevCloseTime = currCloseTime
	s.shortTermClose[cacheLength-1] = data.Close
	s.longTermClose[cacheLength-1] = data.Close
	s.determineEmaCrossoverIndicator()
	return nil
}

func (s *EmaCrossover) GetIndicatorFeed() chan types.Indicator {
	return s.indicatorChan
}

func (s *EmaCrossover) determineEmaCrossoverIndicator() {
	ind := types.Indicator{
		Price:     s.shortTermClose[cacheLength-1],
		Direction: types.None,
	}
	// make sure the cache is filled before making decisions on incomplete data
	if s.shortTermClose[0] == float64(0) {
		s.lastSend = types.None
		s.indicatorChan <- ind
		return
	}
	// Calculate EMAs
	fastEmaVals := talib.Ema(s.shortTermClose, fastEmaLength)
	slowEmaVals := talib.Ema(s.shortTermClose, slowEmaLength)
	ltFastEmaVals := talib.Ema(s.longTermClose, fastEmaLength)
	ltSlowEmaVals := talib.Ema(s.longTermClose, slowEmaLength)
	// Get current values
	fastEma := fastEmaVals[len(fastEmaVals)-1]
	slowEma := slowEmaVals[len(slowEmaVals)-1]
	ltFastEma := ltFastEmaVals[len(ltFastEmaVals)-1]
	ltSlowEma := ltSlowEmaVals[len(ltSlowEmaVals)-1]
	// Determine buy or sell indicators
	/* Buy conditions (reverse for sell)
	 * Long Term Conditions
	   * FastEma above SlowEma
	   * FastEma Increasing
		 * Gap between FastEma and SlowEma is greater than or equal to 2
	 * Short Term Conditions
	 	 * Price is on positive side of 21-EMA
	   * FastEma Crosses SlowEma
	   * FastEma is 1pt above SlowEma
	*/
	// Determine if long term fast EMA is increasing
	ltIncreasing := ltFastEma > ltFastEmaVals[len(ltFastEmaVals)-2]
	ltGap := absDiff(ltFastEma, ltSlowEma) >= 2
	buyCross := fastEmaVals[len(fastEmaVals)-2] < slowEmaVals[len(slowEmaVals)-2] &&
		fastEma > slowEma
	sellCross := fastEmaVals[len(fastEmaVals)-2] > slowEmaVals[len(slowEmaVals)-2] &&
		fastEma < slowEma
	gapBuy := fastEma-slowEma >= 1
	gapSell := slowEma-fastEma >= 1
	buy := ltFastEma > ltSlowEma &&
		ltIncreasing &&
		ltGap &&
		(buyCross || gapBuy) &&
		ind.Price > slowEma
	sell := ltFastEma < ltSlowEma &&
		!ltIncreasing &&
		ltGap &&
		(sellCross || gapSell) &&
		ind.Price < slowEma
	// Determine close indicators
	/*
	* if EMA's cross in opposite direction
	* if price crosses SlowEma + 2pts
	 */
	closeBuy := s.lastSend == types.Buy &&
		(fastEma < slowEma || ind.Price < slowEma-2)
	closeSell := s.lastSend == types.Sell &&
		(fastEma > slowEma || ind.Price > slowEma+2)
	// set indicator
	if buy {
		ind.Direction = types.Buy
	} else if sell {
		ind.Direction = types.Sell
	} else if closeBuy || closeSell {
		ind.Direction = types.Close
	}
	// send it on its way
	s.lastSend = ind.Direction
	s.indicatorChan <- ind
}

func absDiff(a, b float64) float64 {
	if a > b {
		return a - b
	}
	return b - a
}
