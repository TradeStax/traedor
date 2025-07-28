package strategy

import (
	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/strategy/types"
)

type SmaStrategy struct {
	config        *types.Config
	dataCache     []datafeed.Data
	indicatorChan chan types.Indicator
	shortMA       []float64
	longMA        []float64
	lastSignal    int
}

func NewSmaStrategy(c *types.Config, ic chan types.Indicator) types.IStrategy {
	return &SmaStrategy{
		config:        c,
		dataCache:     make([]datafeed.Data, 25), // Increased for crossover calculation
		indicatorChan: ic,
		shortMA:       make([]float64, 0),
		longMA:        make([]float64, 0),
		lastSignal:    types.None,
	}
}

func (s *SmaStrategy) AddData(data datafeed.Data) error {
	for i := 0; i < len(s.dataCache)-1; i++ {
		s.dataCache[i] = s.dataCache[i+1]
	}
	s.dataCache[len(s.dataCache)-1] = data
	s.determineIndicator() // Remove goroutine - execute synchronously
	return nil
}

func (s *SmaStrategy) GetIndicatorFeed() chan types.Indicator {
	return s.indicatorChan
}

func (s *SmaStrategy) determineIndicator() {
	var ind types.Indicator
	ind.Direction = types.None // Default to no signal
	ind.Price = s.dataCache[len(s.dataCache)-1].Close
	
	if s.dataCache[0].Volume == 0 {
		s.indicatorChan <- ind
		return
	}
	
	// Calculate short (10) and long (20) SMAs
	shortMA := s.calculateSMA(10)
	longMA := s.calculateSMA(20)
	
	// Need enough data for long MA
	if longMA == 0 {
		s.indicatorChan <- ind
		return
	}
	
	// Need at least one previous MA value to detect crossover
	if len(s.shortMA) > 0 && len(s.longMA) > 0 {
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

func (s *SmaStrategy) calculateSMA(period int) float64 {
	if period <= 0 || period > len(s.dataCache) {
		return 0
	}
	
	// Calculate SMA for the specified period
	sum := 0.0
	count := 0
	
	for i := len(s.dataCache) - period; i < len(s.dataCache); i++ {
		if s.dataCache[i].Volume > 0 { // Valid data point
			sum += s.dataCache[i].Close
			count++
		}
	}
	
	if count == 0 {
		return 0
	}
	
	return sum / float64(count)
}
