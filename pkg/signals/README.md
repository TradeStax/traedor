# Signal Generation System

The Traedor signal generation system provides a flexible, pluggable architecture for creating and managing trading signals.

## Architecture Overview

The signal system consists of several key components:

### Core Components

1. **ISignalGenerator Interface** - Defines the contract for all signal generators
2. **BaseSignalGenerator** - Common functionality for signal implementations  
3. **SignalManager** - Manages signal definitions and generators
4. **SignalWorkflow** - Orchestrates signals for trading runs
5. **SignalValidator** - Validates signal definitions and parameters

### Built-in Signal Generators

#### SMA Crossover (`sma_crossover`)
Simple Moving Average crossover strategy that generates signals when short-term MA crosses above/below long-term MA.

**Parameters:**
- `short_period` (int): Period for short moving average (default: 20)
- `long_period` (int): Period for long moving average (default: 50)

#### RSI (`rsi`)
Relative Strength Index oscillator that generates signals at overbought/oversold levels.

**Parameters:**
- `period` (int): RSI calculation period (default: 14)
- `overbought_level` (float): Sell signal threshold (default: 70.0)
- `oversold_level` (float): Buy signal threshold (default: 30.0)

## Creating Custom Signals

### 1. Implement ISignalGenerator

```go
type MyCustomSignal struct {
    BaseSignalGenerator
    // Add your custom fields
}

func (s *MyCustomSignal) ProcessTick(tick datafeed.Data) (Signal, error) {
    // Your signal logic here
    return Signal{
        Type:      SignalBuy,  // or SignalSell, SignalNone
        Strength:  0.8,        // 0.0 to 1.0
        Price:     tick.Last,
        Timestamp: tick.Date,
        Metadata:  map[string]interface{}{
            "custom_data": "value",
        },
    }, nil
}
```

### 2. Register Your Signal

```go
func init() {
    RegisterSignalGenerator("my_signal", func() ISignalGenerator {
        return &MyCustomSignal{
            BaseSignalGenerator: NewBaseSignalGenerator(
                "my_signal",
                "My Custom Signal Description",
            ),
        }
    })
}
```

### 3. Add Validation (Optional)

```go
func (s *MyCustomSignal) ValidateParameters(params map[string]interface{}) error {
    // Validate your parameters
    if period, ok := params["period"].(int); !ok || period <= 0 {
        return fmt.Errorf("period must be a positive integer")
    }
    return nil
}

func (s *MyCustomSignal) GetDefaultParameters() map[string]interface{} {
    return map[string]interface{}{
        "period": 14,
        "threshold": 0.5,
    }
}
```

## Using the Signal Workflow

### API Integration

The signal workflow is integrated into the API system for managing signals:

#### Create Signal Definition
```bash
POST /api/v1/signals
{
  "name": "My SMA Strategy",
  "description": "20/50 SMA crossover",
  "type": "technical",
  "parameters": {
    "short_period": 20,
    "long_period": 50
  },
  "active": true
}
```

#### Validate Signal
```bash
POST /api/v1/signals/validate
{
  "name": "Test Signal",
  "type": "sma_crossover",
  "parameters": {
    "short_period": 10,
    "long_period": 20
  }
}
```

#### Check Compatibility
```bash
POST /api/v1/signals/compatibility
{
  "signal_ids": ["signal1", "signal2", "signal3"]
}
```

### Programmatic Usage

```go
// Create signal workflow
workflow := signals.NewSignalWorkflow(storage)

// Initialize signals for a trading run
err := workflow.InitializeRun("run123", []string{"signal1", "signal2"})

// Get generators for processing
generators, exists := workflow.GetRunGenerators("run123")

// Process tick data
for _, generator := range generators {
    signal, err := generator.ProcessTick(tickData)
    if signal.Type != signals.SignalNone {
        // Process the signal
    }
}

// Cleanup when done
workflow.CleanupRun("run123")
```

## Signal States and Lifecycle

### Signal Types
- `SignalNone` - No action
- `SignalBuy` - Enter long position
- `SignalSell` - Enter short position

### Signal Strength
Signals include a strength value (0.0 to 1.0) indicating confidence:
- 0.0 - 0.3: Weak signal
- 0.3 - 0.7: Moderate signal  
- 0.7 - 1.0: Strong signal

### Metadata
Signals can include custom metadata for analysis:
```go
signal.Metadata = map[string]interface{}{
    "rsi_value": 75.2,
    "sma_short": 4521.5,
    "sma_long": 4519.8,
}
```

## Database Integration

Signal definitions are stored in PostgreSQL with full CRUD operations:

```sql
-- Signal definitions table
CREATE TABLE signal_definitions (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    parameters JSONB,
    active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Generated signals table
CREATE TABLE signals (
    id UUID PRIMARY KEY,
    run_id UUID NOT NULL REFERENCES runs(id),
    signal_definition_id UUID,
    symbol VARCHAR(50) NOT NULL,
    direction INTEGER NOT NULL,
    price DECIMAL(18,8) NOT NULL,
    time TIMESTAMP NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
```

## Testing Signals

### Unit Testing
```go
func TestMySignal(t *testing.T) {
    signal := &MyCustomSignal{}
    err := signal.Initialize(map[string]interface{}{
        "period": 14,
    })
    assert.NoError(t, err)
    
    // Test with sample data
    tick := datafeed.Data{Last: 100.0, Date: time.Now().Unix()}
    result, err := signal.ProcessTick(tick)
    assert.NoError(t, err)
    assert.Equal(t, SignalNone, result.Type)
}
```

### Backtesting
Use the API or CLI to backtest your signals:
```bash
# Start backtest with custom signals
curl -X POST localhost:8080/api/v1/runs \
  -d '{"config": {"signals": ["my_signal_id"]}}'
```

## Best Practices

1. **State Management**: Use the BaseSignalGenerator for common functionality
2. **Parameter Validation**: Always validate parameters in `ValidateParameters()`
3. **Error Handling**: Return meaningful errors for debugging
4. **Performance**: Avoid expensive operations in `ProcessTick()`
5. **Testing**: Write unit tests for your signal logic
6. **Documentation**: Provide clear parameter descriptions

## Advanced Features

### Signal Ensembles
Combine multiple signals for enhanced performance:
```go
type EnsembleSignal struct {
    BaseSignalGenerator
    signals []ISignalGenerator
    weights []float64
}
```

### Machine Learning Integration
Integrate ML models for signal generation:
```go
type MLSignal struct {
    BaseSignalGenerator
    model    MLModel
    features []float64
}
```

### Real-time Signals
For live trading, signals can be processed in real-time:
```go
// Process live data stream
for tick := range liveDataStream {
    signal, err := generator.ProcessTick(tick)
    if signal.Type != SignalNone {
        broker.SendTrade(convertSignalToTrade(signal))
    }
}
```