package signal

import (
	"errors"
	"fmt"
)

// Signal direction constants
type SignalDirection int

const (
	NONE SignalDirection = iota
	BUY
	SELL
)

func (s SignalDirection) String() string {
	switch s {
	case BUY:
		return "BUY"
	case SELL:
		return "SELL"
	default:
		return "NONE"
	}
}

// Signal represents a trading signal
type Signal struct {
	Direction SignalDirection `json:"direction"`
	Price     float64         `json:"price"`
	Timestamp int64           `json:"timestamp"`
}

// SignalMetadata contains metadata about a signal
type SignalMetadata struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// ISignalGenerator defines the interface for signal generators
type ISignalGenerator interface {
	Generate(data []float64) (Signal, error)
	GetMetadata() SignalMetadata
	SetParameters(params map[string]interface{}) error
	Validate() error
}

// Registry manages signal generators
type Registry struct {
	generators map[string]ISignalGenerator
}

// NewRegistry creates a new signal registry
func NewRegistry() *Registry {
	return &Registry{
		generators: make(map[string]ISignalGenerator),
	}
}

// Register adds a signal generator to the registry
func (r *Registry) Register(name string, generator ISignalGenerator) error {
	if generator == nil {
		return errors.New("generator cannot be nil")
	}
	
	if _, exists := r.generators[name]; exists {
		return fmt.Errorf("signal generator '%s' already registered", name)
	}
	
	r.generators[name] = generator
	return nil
}

// Get retrieves a signal generator by name
func (r *Registry) Get(name string) (ISignalGenerator, error) {
	generator, exists := r.generators[name]
	if !exists {
		return nil, fmt.Errorf("signal generator '%s' not found", name)
	}
	return generator, nil
}

// List returns all registered signal generators
func (r *Registry) List() []string {
	names := make([]string, 0, len(r.generators))
	for name := range r.generators {
		names = append(names, name)
	}
	return names
}

// GetAll returns all registered signal generators with their metadata
func (r *Registry) GetAll() map[string]SignalMetadata {
	result := make(map[string]SignalMetadata)
	for name, generator := range r.generators {
		result[name] = generator.GetMetadata()
	}
	return result
}

// ValidateSignal validates a signal for correctness
func ValidateSignal(signal Signal) error {
	if signal.Direction < NONE || signal.Direction > SELL {
		return errors.New("invalid signal direction")
	}
	
	if signal.Price < 0 {
		return errors.New("price cannot be negative")
	}
	
	if signal.Timestamp < 0 {
		return errors.New("timestamp cannot be negative")
	}
	
	return nil
}

// SimpleMovingAverageSignal generates signals based on SMA crossover
type SimpleMovingAverageSignal struct {
	shortPeriod int
	longPeriod  int
}

// NewSimpleMovingAverageSignal creates a new SMA signal generator
func NewSimpleMovingAverageSignal(shortPeriod, longPeriod int) *SimpleMovingAverageSignal {
	return &SimpleMovingAverageSignal{
		shortPeriod: shortPeriod,
		longPeriod:  longPeriod,
	}
}

func (s *SimpleMovingAverageSignal) Generate(prices []float64) (Signal, error) {
	if len(prices) < s.longPeriod {
		return Signal{Direction: NONE}, nil
	}
	
	// Calculate short SMA
	shortSum := 0.0
	for i := len(prices) - s.shortPeriod; i < len(prices); i++ {
		shortSum += prices[i]
	}
	shortSMA := shortSum / float64(s.shortPeriod)
	
	// Calculate long SMA
	longSum := 0.0
	for i := len(prices) - s.longPeriod; i < len(prices); i++ {
		longSum += prices[i]
	}
	longSMA := longSum / float64(s.longPeriod)
	
	direction := NONE
	if shortSMA > longSMA {
		direction = BUY
	} else if shortSMA < longSMA {
		direction = SELL
	}
	
	return Signal{
		Direction: direction,
		Price:     prices[len(prices)-1],
		Timestamp: 1234567890,
	}, nil
}

func (s *SimpleMovingAverageSignal) GetMetadata() SignalMetadata {
	return SignalMetadata{
		Name:        "Simple Moving Average",
		Description: "Generates signals based on SMA crossover",
		Type:        "sma",
		Parameters: map[string]interface{}{
			"short_period": s.shortPeriod,
			"long_period":  s.longPeriod,
		},
	}
}

func (s *SimpleMovingAverageSignal) SetParameters(params map[string]interface{}) error {
	if shortPeriod, ok := params["short_period"].(int); ok {
		s.shortPeriod = shortPeriod
	}
	if longPeriod, ok := params["long_period"].(int); ok {
		s.longPeriod = longPeriod
	}
	return nil
}

func (s *SimpleMovingAverageSignal) Validate() error {
	if s.shortPeriod <= 0 || s.longPeriod <= 0 {
		return errors.New("periods must be positive")
	}
	if s.shortPeriod >= s.longPeriod {
		return errors.New("short period must be less than long period")
	}
	return nil
}

// RSISignal generates signals based on RSI
type RSISignal struct {
	period     int
	oversold   float64
	overbought float64
}

// NewRSISignal creates a new RSI signal generator
func NewRSISignal(period int, oversold, overbought float64) *RSISignal {
	return &RSISignal{
		period:     period,
		oversold:   oversold,
		overbought: overbought,
	}
}

func (r *RSISignal) Generate(prices []float64) (Signal, error) {
	if len(prices) < r.period+1 {
		return Signal{Direction: NONE}, nil
	}
	
	// Calculate RSI using the standard formula
	gains := 0.0
	losses := 0.0
	
	// Calculate gains and losses for the period
	for i := len(prices) - r.period; i < len(prices); i++ {
		if i == 0 {
			continue // Skip first price as we need previous price for change
		}
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains += change
		} else if change < 0 {
			losses += -change
		}
	}
	
	avgGain := gains / float64(r.period)
	avgLoss := losses / float64(r.period)
	
	var rsi float64
	if avgLoss == 0 {
		rsi = 100 // No losses means RSI is 100
	} else {
		rs := avgGain / avgLoss
		rsi = 100 - (100 / (1 + rs))
	}
	
	direction := NONE
	if rsi < r.oversold {
		direction = BUY
	} else if rsi > r.overbought {
		direction = SELL
	}
	
	return Signal{
		Direction: direction,
		Price:     prices[len(prices)-1],
		Timestamp: 1234567890,
	}, nil
}

func (r *RSISignal) GetMetadata() SignalMetadata {
	return SignalMetadata{
		Name:        "RSI",
		Description: "Generates signals based on Relative Strength Index",
		Type:        "rsi",
		Parameters: map[string]interface{}{
			"period":     r.period,
			"oversold":   r.oversold,
			"overbought": r.overbought,
		},
	}
}

func (r *RSISignal) SetParameters(params map[string]interface{}) error {
	if period, ok := params["period"].(int); ok {
		r.period = period
	}
	if oversold, ok := params["oversold"].(float64); ok {
		r.oversold = oversold
	}
	if overbought, ok := params["overbought"].(float64); ok {
		r.overbought = overbought
	}
	return nil
}

func (r *RSISignal) Validate() error {
	if r.period <= 0 {
		return errors.New("period must be positive")
	}
	if r.oversold >= r.overbought {
		return errors.New("oversold threshold must be less than overbought threshold")
	}
	return nil
}