package signal

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSignalRegistry_RegisterAndGet(t *testing.T) {
	registry := NewRegistry()
	
	// Create a test signal generator
	testGenerator := &TestSignalGenerator{
		name: "test-signal",
		description: "A test signal for unit testing",
	}
	
	// Register the signal
	err := registry.Register("test-signal", testGenerator)
	require.NoError(t, err)
	
	// Get the signal back
	generator, err := registry.Get("test-signal")
	require.NoError(t, err)
	assert.Equal(t, testGenerator, generator)
	
	// Test getting non-existent signal
	_, err = registry.Get("non-existent")
	assert.Error(t, err)
}

func TestSignalRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewRegistry()
	
	generator1 := &TestSignalGenerator{name: "duplicate"}
	generator2 := &TestSignalGenerator{name: "duplicate"}
	
	// Register first signal
	err := registry.Register("duplicate", generator1)
	require.NoError(t, err)
	
	// Try to register duplicate
	err = registry.Register("duplicate", generator2)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already registered")
}

func TestSignalRegistry_List(t *testing.T) {
	registry := NewRegistry()
	
	// Register multiple signals
	signals := map[string]ISignalGenerator{
		"signal1": &TestSignalGenerator{name: "signal1"},
		"signal2": &TestSignalGenerator{name: "signal2"},
		"signal3": &TestSignalGenerator{name: "signal3"},
	}
	
	for name, generator := range signals {
		registry.Register(name, generator)
	}
	
	// List all signals
	registered := registry.List()
	assert.Len(t, registered, 3)
	
	// Check all signals are present
	for name := range signals {
		assert.Contains(t, registered, name)
	}
}

func TestSignalValidation(t *testing.T) {
	tests := []struct {
		name      string
		signal    Signal
		expectErr bool
	}{
		{
			name: "valid buy signal",
			signal: Signal{
				Direction: BUY,
				Price:     100.0,
				Timestamp: 1234567890,
			},
			expectErr: false,
		},
		{
			name: "valid sell signal",
			signal: Signal{
				Direction: SELL,
				Price:     100.0,
				Timestamp: 1234567890,
			},
			expectErr: false,
		},
		{
			name: "valid none signal",
			signal: Signal{
				Direction: NONE,
				Price:     100.0,
				Timestamp: 1234567890,
			},
			expectErr: false,
		},
		{
			name: "invalid direction",
			signal: Signal{
				Direction: 999,
				Price:     100.0,
				Timestamp: 1234567890,
			},
			expectErr: true,
		},
		{
			name: "invalid price",
			signal: Signal{
				Direction: BUY,
				Price:     -100.0,
				Timestamp: 1234567890,
			},
			expectErr: true,
		},
		{
			name: "invalid timestamp",
			signal: Signal{
				Direction: BUY,
				Price:     100.0,
				Timestamp: -1,
			},
			expectErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSignal(tt.signal)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSimpleMovingAverageSignal(t *testing.T) {
	// Create SMA signal generator with short periods for testing
	smaGenerator := NewSimpleMovingAverageSignal(3, 5) // 3-period vs 5-period SMA
	
	// Test with insufficient data
	signal, err := smaGenerator.Generate([]float64{100, 101})
	require.NoError(t, err)
	assert.Equal(t, NONE, signal.Direction)
	
	// Test with upward trend (short SMA > long SMA)
	prices := []float64{100, 101, 102, 103, 104, 105, 106}
	signal, err = smaGenerator.Generate(prices)
	require.NoError(t, err)
	assert.Equal(t, BUY, signal.Direction)
	assert.Equal(t, 106.0, signal.Price) // Latest price
	
	// Test with downward trend (short SMA < long SMA)
	prices = []float64{106, 105, 104, 103, 102, 101, 100}
	signal, err = smaGenerator.Generate(prices)
	require.NoError(t, err)
	assert.Equal(t, SELL, signal.Direction)
	assert.Equal(t, 100.0, signal.Price) // Latest price
}

func TestRSISignal(t *testing.T) {
	// Create RSI signal generator
	rsiGenerator := NewRSISignal(14, 30, 70) // 14-period RSI, oversold at 30, overbought at 70
	
	// Test with insufficient data
	signal, err := rsiGenerator.Generate([]float64{100, 101, 102})
	require.NoError(t, err)
	assert.Equal(t, NONE, signal.Direction)
	
	// Create a price series that should generate oversold condition
	prices := make([]float64, 20)
	for i := 0; i < 20; i++ {
		prices[i] = 100.0 - float64(i) // Declining prices
	}
	
	signal, err = rsiGenerator.Generate(prices)
	require.NoError(t, err)
	// RSI should be low (oversold), so expect BUY signal
	assert.Equal(t, BUY, signal.Direction)
	
	// Create a price series that should generate overbought condition
	for i := 0; i < 20; i++ {
		prices[i] = 80.0 + float64(i) // Rising prices
	}
	
	signal, err = rsiGenerator.Generate(prices)
	require.NoError(t, err)
	// RSI should be high (overbought), so expect SELL signal
	assert.Equal(t, SELL, signal.Direction)
}

// Test signal generator for unit tests
type TestSignalGenerator struct {
	name        string
	description string
}

func (t *TestSignalGenerator) Generate(prices []float64) (Signal, error) {
	if len(prices) == 0 {
		return Signal{Direction: NONE}, nil
	}
	
	// Simple test logic: buy if last price > 100, sell if < 100
	lastPrice := prices[len(prices)-1]
	direction := NONE
	if lastPrice > 100 {
		direction = BUY
	} else if lastPrice < 100 {
		direction = SELL
	}
	
	return Signal{
		Direction: direction,
		Price:     lastPrice,
		Timestamp: 1234567890,
	}, nil
}

func (t *TestSignalGenerator) GetMetadata() SignalMetadata {
	return SignalMetadata{
		Name:        t.name,
		Description: t.description,
		Type:        "test",
		Parameters:  map[string]interface{}{},
	}
}

func (t *TestSignalGenerator) SetParameters(params map[string]interface{}) error {
	return nil
}

func (t *TestSignalGenerator) Validate() error {
	return nil
}