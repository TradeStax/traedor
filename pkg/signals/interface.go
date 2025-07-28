package signals

import (
	"github.com/tradestax/traedor/pkg/datafeed"
)

type SignalType int

const (
	SignalNone SignalType = iota
	SignalBuy
	SignalSell
)

type Signal struct {
	Type      SignalType
	Strength  float64 // 0.0 to 1.0
	Price     float64
	Timestamp int64
	Metadata  map[string]interface{}
}

type ISignalGenerator interface {
	// Initialize the signal generator with parameters
	Initialize(params map[string]interface{}) error
	
	// Process tick data and generate a signal
	ProcessTick(tick datafeed.Data) (Signal, error)
	
	// Get the name of the signal generator
	GetName() string
	
	// Get the description of the signal generator
	GetDescription() string
	
	// Get the default parameters for this signal generator
	GetDefaultParameters() map[string]interface{}
	
	// Validate parameters before initialization
	ValidateParameters(params map[string]interface{}) error
	
	// Reset internal state
	Reset() error
}

type SignalGeneratorFactory func() ISignalGenerator

var signalGenerators = make(map[string]SignalGeneratorFactory)

func RegisterSignalGenerator(name string, factory SignalGeneratorFactory) {
	signalGenerators[name] = factory
}

func GetSignalGenerator(name string) (ISignalGenerator, bool) {
	factory, exists := signalGenerators[name]
	if !exists {
		return nil, false
	}
	return factory(), true
}

func GetAvailableSignalGenerators() []string {
	names := make([]string, 0, len(signalGenerators))
	for name := range signalGenerators {
		names = append(names, name)
	}
	return names
}