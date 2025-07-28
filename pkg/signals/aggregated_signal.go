package signals

import (
	"fmt"

	"github.com/tradestax/traedor/pkg/aggregator"
	"github.com/tradestax/traedor/pkg/datafeed"
)

type AggregatedSignal struct {
	innerSignal     ISignalGenerator
	aggregator      *aggregator.TimeAggregator
	intervalMinutes int
	lastSignal      Signal
	barReady        bool
	currentBar      *aggregator.OHLC
}

func NewAggregatedSignal(innerSignal ISignalGenerator, intervalMinutes int) *AggregatedSignal {
	fmt.Printf("AggregatedSignal: Creating %d-minute aggregated signal for %s\n", intervalMinutes, innerSignal.GetName())
	
	as := &AggregatedSignal{
		innerSignal:     innerSignal,
		intervalMinutes: intervalMinutes,
		lastSignal: Signal{
			Type:     SignalNone,
			Strength: 0.0,
		},
	}

	// Create aggregator with callback
	as.aggregator = aggregator.NewTimeAggregator(intervalMinutes, func(bar aggregator.OHLC) {
		fmt.Printf("AggregatedSignal: Bar completed callback - OHLC: %.2f/%.2f/%.2f/%.2f\n", 
			bar.Open, bar.High, bar.Low, bar.Close)
		as.currentBar = &bar
		as.barReady = true
	})

	return as
}

func (as *AggregatedSignal) Initialize(params map[string]interface{}) error {
	// Initialize the inner signal with the same parameters
	return as.innerSignal.Initialize(params)
}

func (as *AggregatedSignal) ProcessTick(tick datafeed.Data) (Signal, error) {
	// Feed tick to aggregator
	as.aggregator.ProcessTick(tick)

	// If we have a completed bar, process it
	if as.barReady && as.currentBar != nil {
		fmt.Printf("AggregatedSignal: Processing completed %d-minute bar - OHLC: %.2f/%.2f/%.2f/%.2f\n", 
			as.intervalMinutes, as.currentBar.Open, as.currentBar.High, as.currentBar.Low, as.currentBar.Close)
		// Convert OHLC bar to datafeed.Data for the inner signal
		barData := datafeed.Data{
			Date:   as.currentBar.Timestamp,
			Open:   as.currentBar.Open,
			High:   as.currentBar.High,
			Low:    as.currentBar.Low,
			Close:  as.currentBar.Close,
			Volume: as.currentBar.Volume,
		}

		// Process the bar through the inner signal
		signal, err := as.innerSignal.ProcessTick(barData)
		if err != nil {
			return signal, err
		}

		// Add aggregation metadata
		if signal.Metadata == nil {
			signal.Metadata = make(map[string]interface{})
		}
		signal.Metadata["aggregation_interval"] = as.intervalMinutes
		signal.Metadata["bar_start_time"] = as.currentBar.StartTime
		signal.Metadata["bar_end_time"] = as.currentBar.EndTime

		as.lastSignal = signal
		as.barReady = false
		as.currentBar = nil

		return signal, nil
	}

	// Return no signal if no new bar is ready to prevent signal repetition
	return Signal{
		Type:      SignalNone,
		Strength:  0.0,
		Price:     tick.Close,
		Timestamp: tick.Date,
		Metadata:  make(map[string]interface{}),
	}, nil
}

func (as *AggregatedSignal) GetName() string {
	return fmt.Sprintf("%s_%dm", as.innerSignal.GetName(), as.intervalMinutes)
}

func (as *AggregatedSignal) GetDescription() string {
	return fmt.Sprintf("%s (aggregated to %d minute bars)", as.innerSignal.GetDescription(), as.intervalMinutes)
}

func (as *AggregatedSignal) GetDefaultParameters() map[string]interface{} {
	params := as.innerSignal.GetDefaultParameters()
	params["aggregation_interval"] = as.intervalMinutes
	return params
}

func (as *AggregatedSignal) ValidateParameters(params map[string]interface{}) error {
	return as.innerSignal.ValidateParameters(params)
}

func (as *AggregatedSignal) Reset() error {
	as.aggregator.Reset()
	as.barReady = false
	as.currentBar = nil
	as.lastSignal = Signal{
		Type:     SignalNone,
		Strength: 0.0,
	}
	return as.innerSignal.Reset()
}

// Flush completes any pending bar and processes it through the inner signal
func (as *AggregatedSignal) Flush() (Signal, error) {
	fmt.Printf("AggregatedSignal: Flushing final bar\n")
	
	// Flush any pending bar from the aggregator
	as.aggregator.Flush()
	
	// If we have a completed bar after flushing, process it
	if as.barReady && as.currentBar != nil {
		fmt.Printf("AggregatedSignal: Processing flushed %d-minute bar - OHLC: %.2f/%.2f/%.2f/%.2f\n", 
			as.intervalMinutes, as.currentBar.Open, as.currentBar.High, as.currentBar.Low, as.currentBar.Close)
		
		// Convert OHLC bar to datafeed.Data for the inner signal
		barData := datafeed.Data{
			Date:   as.currentBar.Timestamp,
			Open:   as.currentBar.Open,
			High:   as.currentBar.High,
			Low:    as.currentBar.Low,
			Close:  as.currentBar.Close,
			Volume: as.currentBar.Volume,
		}

		// Process the bar through the inner signal
		signal, err := as.innerSignal.ProcessTick(barData)
		if err != nil {
			return signal, err
		}

		// Add aggregation metadata
		if signal.Metadata == nil {
			signal.Metadata = make(map[string]interface{})
		}
		signal.Metadata["aggregation_interval"] = as.intervalMinutes
		signal.Metadata["bar_start_time"] = as.currentBar.StartTime
		signal.Metadata["bar_end_time"] = as.currentBar.EndTime

		as.lastSignal = signal
		as.barReady = false
		as.currentBar = nil

		return signal, nil
	}
	
	// Return no signal if no bar was pending
	return Signal{
		Type:      SignalNone,
		Strength:  0.0,
		Metadata:  make(map[string]interface{}),
	}, nil
}