package strategy

import (
	"fmt"
	"sync"

	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/signals"
	"github.com/tradestax/traedor/pkg/strategy/types"
)

type SignalAdapterStrategy struct {
	indicatorFeed   chan types.Indicator
	signalGenerator signals.ISignalGenerator
	symbol          string
	ignoreNone      bool
	mu              sync.Mutex
	tickCount       int
}

func NewSignalAdapterStrategy(config *types.Config, indicatorFeed chan types.Indicator) types.IStrategy {
	return &SignalAdapterStrategy{
		indicatorFeed: indicatorFeed,
		symbol:        config.Symbol,
		ignoreNone:    config.IgnoreNone,
	}
}

func (s *SignalAdapterStrategy) Initialize(signalName string, params map[string]interface{}) error {
	// Check if this signal has aggregation parameters
	var generator signals.ISignalGenerator
	if aggregationInterval, hasAggregation := params["aggregation_interval"]; hasAggregation {
		// Create aggregated signal
		intervalMinutes, ok := aggregationInterval.(float64)
		if !ok {
			return fmt.Errorf("invalid aggregation_interval type")
		}

		baseGenerator, exists := signals.GetSignalGenerator(signalName)
		if !exists {
			return fmt.Errorf("signal generator '%s' not found", signalName)
		}

		// Create aggregated wrapper
		generator = signals.NewAggregatedSignal(baseGenerator, int(intervalMinutes))
	} else {
		// Create regular signal
		var exists bool
		generator, exists = signals.GetSignalGenerator(signalName)
		if !exists {
			return fmt.Errorf("signal generator '%s' not found", signalName)
		}
	}

	if err := generator.Initialize(params); err != nil {
		return fmt.Errorf("failed to initialize signal generator: %w", err)
	}

	s.signalGenerator = generator
	fmt.Printf("SignalAdapter: Successfully initialized signal generator '%s' with params: %+v\n", signalName, params)
	return nil
}

func (s *SignalAdapterStrategy) AddData(data datafeed.Data) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.signalGenerator == nil {
		fmt.Printf("SignalAdapter: ERROR - signal generator not initialized\n")
		return fmt.Errorf("signal generator not initialized")
	}

	s.tickCount++
	
	// Log first few ticks and every 10000th tick
	if s.tickCount <= 10 || s.tickCount%10000 == 0 {
		fmt.Printf("SignalAdapter: Processing tick #%d - Price: %.2f\n", s.tickCount, data.Close)
	}

	// Process tick through signal generator
	signal, err := s.signalGenerator.ProcessTick(data)
	if err != nil {
		return fmt.Errorf("failed to process tick: %w", err)
	}

	// Convert signal to indicator
	indicator := s.signalToIndicator(signal, data)

	// Log when we get actual signals
	if signal.Type != signals.SignalNone {
		fmt.Printf("SignalAdapter: Generated signal %v at price %.2f\n", signal.Type, data.Close)
	}

	// Send indicator if not ignoring None signals or if signal is not None
	if !s.ignoreNone || indicator.Direction != types.None {
		select {
		case s.indicatorFeed <- indicator:
		default:
			// Channel full, skip
			fmt.Printf("SignalAdapter: WARNING - Indicator channel full, dropping signal\n")
		}
	}

	return nil
}

func (s *SignalAdapterStrategy) GetIndicatorFeed() chan types.Indicator {
	return s.indicatorFeed
}

// Flush completes any pending aggregated bars and processes final signals
func (s *SignalAdapterStrategy) Flush() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.signalGenerator == nil {
		return nil
	}
	
	fmt.Printf("SignalAdapter: Flushing final signals\n")
	
	// Check if the signal generator is an aggregated signal that needs flushing
	if aggregatedSignal, ok := s.signalGenerator.(interface{ Flush() (signals.Signal, error) }); ok {
		signal, err := aggregatedSignal.Flush()
		if err != nil {
			return fmt.Errorf("failed to flush aggregated signal: %w", err)
		}
		
		// Convert signal to indicator
		if signal.Type != signals.SignalNone {
			fmt.Printf("SignalAdapter: Generated final signal %v\n", signal.Type)
			
			// Create a mock data point for the final signal
			mockData := datafeed.Data{
				Date:  signal.Timestamp,
				Close: signal.Price,
			}
			
			indicator := s.signalToIndicator(signal, mockData)
			
			// Send final indicator
			if !s.ignoreNone || indicator.Direction != types.None {
				select {
				case s.indicatorFeed <- indicator:
				default:
					fmt.Printf("SignalAdapter: WARNING - Indicator channel full, dropping final signal\n")
				}
			}
		}
	}
	
	return nil
}

func (s *SignalAdapterStrategy) signalToIndicator(signal signals.Signal, data datafeed.Data) types.Indicator {
	var direction int
	
	switch signal.Type {
	case signals.SignalBuy:
		direction = types.Buy
	case signals.SignalSell:
		direction = types.Sell
	default:
		direction = types.None
	}

	return types.Indicator{
		Direction: direction,
		Price:     data.Close,
		Time:      data.Date,
	}
}

// Factory function to create signal adapter strategies
func NewSignalStrategy(signalName string, params map[string]interface{}) func(*types.Config, chan types.Indicator) types.IStrategy {
	return func(config *types.Config, indicatorFeed chan types.Indicator) types.IStrategy {
		strategy := &SignalAdapterStrategy{
			indicatorFeed: indicatorFeed,
			symbol:        config.Symbol,
			ignoreNone:    config.IgnoreNone,
		}
		
		// Initialize with the signal name and parameters from config
		if config.Params.Values != nil {
			// Convert params from config
			convertedParams := make(map[string]interface{})
			for k, v := range params {
				convertedParams[k] = v
			}
			strategy.Initialize(signalName, convertedParams)
		} else {
			strategy.Initialize(signalName, params)
		}
		
		return strategy
	}
}

// Factory function for strategy registration
func NewSignalAdapterStrategyFactory(config *types.Config, indicatorFeed chan types.Indicator) types.IStrategy {
	strategy := &SignalAdapterStrategy{
		indicatorFeed: indicatorFeed,
		symbol:        config.Symbol,
		ignoreNone:    config.IgnoreNone,
	}
	
	// Get signal type from config
	if len(config.Params.Values) > 0 {
		signalType := config.Params.Values[0]
		signalParams := config.SignalParams
		if signalParams == nil {
			signalParams = make(map[string]interface{})
		}
		
		// Initialize the signal generator
		if err := strategy.Initialize(signalType, signalParams); err != nil {
			// Log the error instead of panicking to debug the issue
			fmt.Printf("ERROR: Failed to initialize signal adapter strategy: %v\n", err)
			fmt.Printf("Signal type: %s, Signal params: %+v\n", signalType, signalParams)
			// Return a no-op strategy instead of panicking
			return &SignalAdapterStrategy{
				indicatorFeed: indicatorFeed,
				symbol:        config.Symbol,
				ignoreNone:    true,
			}
		}
	} else {
		fmt.Printf("ERROR: Signal adapter strategy requires signal type in Params.Values[0]\n")
		fmt.Printf("Config: %+v\n", config)
		// Return a no-op strategy instead of panicking
		return &SignalAdapterStrategy{
			indicatorFeed: indicatorFeed,
			symbol:        config.Symbol,
			ignoreNone:    true,
		}
	}
	
	return strategy
}