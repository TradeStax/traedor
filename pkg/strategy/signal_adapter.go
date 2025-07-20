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
}

func NewSignalAdapterStrategy(config *types.Config, indicatorFeed chan types.Indicator) types.IStrategy {
	return &SignalAdapterStrategy{
		indicatorFeed: indicatorFeed,
		symbol:        config.Symbol,
		ignoreNone:    config.IgnoreNone,
	}
}

func (s *SignalAdapterStrategy) Initialize(signalName string, params map[string]interface{}) error {
	generator, exists := signals.GetSignalGenerator(signalName)
	if !exists {
		return fmt.Errorf("signal generator '%s' not found", signalName)
	}

	if err := generator.Initialize(params); err != nil {
		return fmt.Errorf("failed to initialize signal generator: %w", err)
	}

	s.signalGenerator = generator
	return nil
}

func (s *SignalAdapterStrategy) AddData(data datafeed.Data) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.signalGenerator == nil {
		return fmt.Errorf("signal generator not initialized")
	}

	// Process tick through signal generator
	signal, err := s.signalGenerator.ProcessTick(data)
	if err != nil {
		return fmt.Errorf("failed to process tick: %w", err)
	}

	// Convert signal to indicator
	indicator := s.signalToIndicator(signal, data)

	// Send indicator if not ignoring None signals or if signal is not None
	if !s.ignoreNone || indicator.Direction != types.None {
		select {
		case s.indicatorFeed <- indicator:
		default:
			// Channel full, skip
		}
	}

	return nil
}

func (s *SignalAdapterStrategy) GetIndicatorFeed() chan types.Indicator {
	return s.indicatorFeed
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