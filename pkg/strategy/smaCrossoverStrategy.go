package strategy

import (
	"log"
	"time"
	
	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/strategy/types"
)

type OHLC struct {
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	Timestamp time.Time
}

type SmaCrossoverStrategy struct {
	config        *types.Config
	dataCache     []datafeed.Data
	indicatorChan chan types.Indicator
	shortPeriod   int
	longPeriod    int
	shortMA       []float64
	longMA        []float64
	lastSignal    int
	
	// Aggregation fields
	barIntervalMinutes int
	currentBar         *OHLC
	lastBarTime        time.Time
	completedBars      []OHLC
}

func NewSmaCrossoverStrategy(c *types.Config, ic chan types.Indicator) types.IStrategy {
	log.Printf("Creating SMA crossover strategy with 5/10 periods and 5-minute bars")
	return &SmaCrossoverStrategy{
		config:        c,
		dataCache:     make([]datafeed.Data, 50), // Enough for long MA calculation
		indicatorChan: ic,
		shortPeriod:   5,  // Default short period
		longPeriod:    10, // Default long period
		shortMA:       make([]float64, 0),
		longMA:        make([]float64, 0),
		lastSignal:    types.None,
		
		// Use 5-minute bars for aggregation
		barIntervalMinutes: 5,
		completedBars:      make([]OHLC, 0, 50),
	}
}

func (s *SmaCrossoverStrategy) AddData(data datafeed.Data) error {
	// Convert Unix timestamp to time
	tickTime := time.Unix(data.Date, 0)
	
	// Aggregate ticks into bars
	if s.processTickIntoBar(data, tickTime) {
		// A bar was completed, process it
		s.determineIndicator()
	}
	
	return nil
}

func (s *SmaCrossoverStrategy) GetIndicatorFeed() chan types.Indicator {
	return s.indicatorChan
}

func (s *SmaCrossoverStrategy) processTickIntoBar(data datafeed.Data, tickTime time.Time) bool {
	// Truncate time to bar interval
	barTime := tickTime.Truncate(time.Duration(s.barIntervalMinutes) * time.Minute)
	
	// Check if this is a new bar
	if s.currentBar == nil || !barTime.Equal(s.lastBarTime) {
		// Complete the previous bar if it exists
		if s.currentBar != nil {
			s.completedBars = append(s.completedBars, *s.currentBar)
			log.Printf("Completed bar %d at %v, close: %.2f", len(s.completedBars), s.currentBar.Timestamp, s.currentBar.Close)
			
			// Keep only the last 50 bars for memory efficiency
			if len(s.completedBars) > 50 {
				s.completedBars = s.completedBars[1:]
			}
		}
		
		// Start a new bar
		s.currentBar = &OHLC{
			Open:      data.Close,
			High:      data.Close,
			Low:       data.Close,
			Close:     data.Close,
			Volume:    data.Volume,
			Timestamp: barTime,
		}
		s.lastBarTime = barTime
		
		// Return true only if we completed a bar (not for the first bar)
		return len(s.completedBars) > 0
	} else {
		// Update current bar with this tick
		if data.Close > s.currentBar.High {
			s.currentBar.High = data.Close
		}
		if data.Close < s.currentBar.Low {
			s.currentBar.Low = data.Close
		}
		s.currentBar.Close = data.Close
		s.currentBar.Volume += data.Volume
	}
	
	return false
}

func (s *SmaCrossoverStrategy) determineIndicator() {
	var ind types.Indicator
	ind.Direction = types.None // Default to no signal
	
	// Use the last completed bar's close price
	if len(s.completedBars) == 0 {
		s.indicatorChan <- ind
		return
	}
	
	lastBar := s.completedBars[len(s.completedBars)-1]
	ind.Price = lastBar.Close
	
	// Need enough bars for long MA calculation
	if len(s.completedBars) < s.longPeriod {
		s.indicatorChan <- ind
		return
	}
	
	// Calculate current short and long MAs from completed bars
	shortMA := s.calculateSMAFromBars(s.shortPeriod)
	longMA := s.calculateSMAFromBars(s.longPeriod)
	
	// Need at least one previous MA value to detect crossover
	if len(s.shortMA) > 0 && len(s.longMA) > 0 && shortMA != 0 && longMA != 0 {
		prevShortMA := s.shortMA[len(s.shortMA)-1]
		prevLongMA := s.longMA[len(s.longMA)-1]
		
		// Detect crossovers
		if prevShortMA <= prevLongMA && shortMA > longMA {
			// Bullish crossover: short MA crosses above long MA
			ind.Direction = types.Buy
			s.lastSignal = types.Buy
		} else if prevShortMA >= prevLongMA && shortMA < longMA {
			// Bearish crossover: short MA crosses below long MA
			ind.Direction = types.Sell
			s.lastSignal = types.Sell
		}
		// Note: No automatic close signals to prevent overtrading
	}
	
	// Store current MAs
	s.shortMA = append(s.shortMA, shortMA)
	s.longMA = append(s.longMA, longMA)
	
	// Keep only recent MA values (last 10)
	if len(s.shortMA) > 10 {
		s.shortMA = s.shortMA[1:]
	}
	if len(s.longMA) > 10 {
		s.longMA = s.longMA[1:]
	}
	
	s.indicatorChan <- ind
}

func (s *SmaCrossoverStrategy) calculateSMAFromBars(period int) float64 {
	if period <= 0 || period > len(s.completedBars) {
		return 0
	}
	
	// Calculate SMA for the specified period from completed bars
	sum := 0.0
	count := 0
	
	for i := len(s.completedBars) - period; i < len(s.completedBars); i++ {
		sum += s.completedBars[i].Close
		count++
	}
	
	if count == 0 {
		return 0
	}
	
	return sum / float64(count)
}