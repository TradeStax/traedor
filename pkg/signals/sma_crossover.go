package signals

import (
	"fmt"

	"github.com/tradestax/traedor/pkg/datafeed"
)

type SMACrossoverSignal struct {
	BaseSignalGenerator
	shortPeriod   int
	longPeriod    int
	shortMA       []float64
	longMA        []float64
	priceHistory  []float64
	lastSignal    SignalType
}

func init() {
	RegisterSignalGenerator("sma_crossover", func() ISignalGenerator {
		return &SMACrossoverSignal{
			BaseSignalGenerator: NewBaseSignalGenerator(
				"sma_crossover",
				"Simple Moving Average Crossover Signal Generator",
			),
		}
	})
}

func (s *SMACrossoverSignal) Initialize(params map[string]interface{}) error {
	if err := s.BaseSignalGenerator.Initialize(params); err != nil {
		return err
	}

	var err error
	s.shortPeriod, err = s.GetIntParameter("short_period", 20)
	if err != nil {
		return err
	}

	s.longPeriod, err = s.GetIntParameter("long_period", 50)
	if err != nil {
		return err
	}

	// Validate parameters
	if s.shortPeriod <= 0 {
		return fmt.Errorf("short_period must be positive, got %d", s.shortPeriod)
	}
	if s.longPeriod <= 0 {
		return fmt.Errorf("long_period must be positive, got %d", s.longPeriod)
	}
	if s.shortPeriod >= s.longPeriod {
		return fmt.Errorf("short_period (%d) must be less than long_period (%d)", s.shortPeriod, s.longPeriod)
	}

	s.shortMA = make([]float64, 0, s.shortPeriod)
	s.longMA = make([]float64, 0, s.longPeriod)
	s.priceHistory = make([]float64, 0, s.longPeriod)
	s.lastSignal = SignalNone

	return nil
}

func (s *SMACrossoverSignal) ProcessTick(tick datafeed.Data) (Signal, error) {
	// Add price to history
	s.priceHistory = append(s.priceHistory, tick.Close)
	
	// Keep only the needed history
	if len(s.priceHistory) > s.longPeriod {
		s.priceHistory = s.priceHistory[1:]
	}

	signal := Signal{
		Type:      SignalNone,
		Strength:  0.0,
		Price:     tick.Close,
		Timestamp: tick.Date,
		Metadata:  make(map[string]interface{}),
	}

	// Need enough data for long MA
	if len(s.priceHistory) < s.longPeriod {
		// Debug: Log when we don't have enough data yet
		if len(s.priceHistory)%100 == 0 { // Log every 100 ticks to avoid spam
			fmt.Printf("SMA Crossover: Need %d periods, have %d (%.1f%%)\n", 
				s.longPeriod, len(s.priceHistory), 
				float64(len(s.priceHistory))/float64(s.longPeriod)*100)
		}
		return signal, nil
	}

	// Calculate short MA - ensure we have enough history
	if len(s.priceHistory) < s.shortPeriod {
		// Not enough data for short MA calculation yet
		return signal, nil
	}
	
	shortMA := s.calculateMA(s.priceHistory[len(s.priceHistory)-s.shortPeriod:])
	
	// Calculate long MA
	longMA := s.calculateMA(s.priceHistory)

	// Store MAs in metadata
	signal.Metadata["short_ma"] = shortMA
	signal.Metadata["long_ma"] = longMA

	// First time reaching crossover logic - log it
	if len(s.shortMA) == 0 && len(s.longMA) == 0 {
		fmt.Printf("SMA Crossover: First calculation - Short MA: %.2f, Long MA: %.2f\n", shortMA, longMA)
	}

	// Generate signal based on crossover
	if len(s.shortMA) > 0 && len(s.longMA) > 0 {
		prevShortMA := s.shortMA[len(s.shortMA)-1]
		prevLongMA := s.longMA[len(s.longMA)-1]

		// Bullish crossover: short MA crosses above long MA
		if prevShortMA <= prevLongMA && shortMA > longMA {
			fmt.Printf("SMA Crossover: BUY SIGNAL - Short MA %.2f crossed above Long MA %.2f (prev: %.2f/%.2f)\n", 
				shortMA, longMA, prevShortMA, prevLongMA)
			signal.Type = SignalBuy
			signal.Strength = s.calculateStrength(shortMA, longMA)
			s.lastSignal = SignalBuy
		} else if prevShortMA >= prevLongMA && shortMA < longMA {
			// Bearish crossover: short MA crosses below long MA
			fmt.Printf("SMA Crossover: SELL SIGNAL - Short MA %.2f crossed below Long MA %.2f (prev: %.2f/%.2f)\n", 
				shortMA, longMA, prevShortMA, prevLongMA)
			signal.Type = SignalSell
			signal.Strength = s.calculateStrength(longMA, shortMA)
			s.lastSignal = SignalSell
		}
	}

	// Update MA history
	s.shortMA = append(s.shortMA, shortMA)
	s.longMA = append(s.longMA, longMA)
	
	// Keep only recent MA values
	if len(s.shortMA) > 10 {
		s.shortMA = s.shortMA[1:]
	}
	if len(s.longMA) > 10 {
		s.longMA = s.longMA[1:]
	}

	return signal, nil
}

func (s *SMACrossoverSignal) calculateMA(prices []float64) float64 {
	if len(prices) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, price := range prices {
		sum += price
	}
	return sum / float64(len(prices))
}


func (s *SMACrossoverSignal) calculateStrength(ma1, ma2 float64) float64 {
	// Calculate strength based on the difference between MAs
	diff := (ma1 - ma2) / ma2
	if diff < 0 {
		diff = -diff
	}
	
	// Cap strength at 1.0
	strength := diff * 100 // Scale up the difference
	if strength > 1.0 {
		strength = 1.0
	}
	
	return strength
}

func (s *SMACrossoverSignal) GetDefaultParameters() map[string]interface{} {
	return map[string]interface{}{
		"short_period": 20,
		"long_period":  50,
	}
}

func (s *SMACrossoverSignal) ValidateParameters(params map[string]interface{}) error {
	shortPeriod := 20 // default
	longPeriod := 50  // default
	
	// Extract short_period
	if val, ok := params["short_period"]; ok {
		switch v := val.(type) {
		case int:
			shortPeriod = v
		case int64:
			shortPeriod = int(v)
		case float64:
			shortPeriod = int(v)
		default:
			return fmt.Errorf("short_period must be a number")
		}
	}
	
	// Extract long_period
	if val, ok := params["long_period"]; ok {
		switch v := val.(type) {
		case int:
			longPeriod = v
		case int64:
			longPeriod = int(v)
		case float64:
			longPeriod = int(v)
		default:
			return fmt.Errorf("long_period must be a number")
		}
	}
	
	if shortPeriod <= 0 {
		return fmt.Errorf("short_period must be greater than 0")
	}
	
	if longPeriod <= 0 {
		return fmt.Errorf("long_period must be greater than 0")
	}
	
	if shortPeriod >= longPeriod {
		return fmt.Errorf("short_period must be less than long_period")
	}
	
	return nil
}

func (s *SMACrossoverSignal) Reset() error {
	s.shortMA = make([]float64, 0, s.shortPeriod)
	s.longMA = make([]float64, 0, s.longPeriod)
	s.priceHistory = make([]float64, 0, s.longPeriod)
	s.lastSignal = SignalNone
	return nil
}