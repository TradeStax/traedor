package signals

import (
	"fmt"

	"github.com/tradestax/traedor/pkg/datafeed"
)

type RSISignal struct {
	BaseSignalGenerator
	period          int
	overboughtLevel float64
	oversoldLevel   float64
	gains           []float64
	losses          []float64
	priceHistory    []float64
	lastRSI         float64
}

func init() {
	RegisterSignalGenerator("rsi", func() ISignalGenerator {
		return &RSISignal{
			BaseSignalGenerator: NewBaseSignalGenerator(
				"rsi",
				"Relative Strength Index Signal Generator",
			),
		}
	})
}

func (r *RSISignal) Initialize(params map[string]interface{}) error {
	if err := r.BaseSignalGenerator.Initialize(params); err != nil {
		return err
	}

	var err error
	r.period, err = r.GetIntParameter("period", 14)
	if err != nil {
		return err
	}

	r.overboughtLevel, err = r.GetFloatParameter("overbought_level", 70.0)
	if err != nil {
		return err
	}

	r.oversoldLevel, err = r.GetFloatParameter("oversold_level", 30.0)
	if err != nil {
		return err
	}

	r.gains = make([]float64, 0, r.period)
	r.losses = make([]float64, 0, r.period)
	r.priceHistory = make([]float64, 0, r.period+1)
	r.lastRSI = 50.0 // neutral

	return nil
}

func (r *RSISignal) ProcessTick(tick datafeed.Data) (Signal, error) {
	// Add price to history
	r.priceHistory = append(r.priceHistory, tick.Last)

	signal := Signal{
		Type:      SignalNone,
		Strength:  0.0,
		Price:     tick.Last,
		Timestamp: tick.Date,
		Metadata:  make(map[string]interface{}),
	}

	// Need at least 2 prices to calculate change
	if len(r.priceHistory) < 2 {
		return signal, nil
	}

	// Calculate price change
	change := r.priceHistory[len(r.priceHistory)-1] - r.priceHistory[len(r.priceHistory)-2]
	
	gain := 0.0
	loss := 0.0
	if change > 0 {
		gain = change
	} else {
		loss = -change
	}

	r.gains = append(r.gains, gain)
	r.losses = append(r.losses, loss)

	// Keep only the needed history
	if len(r.priceHistory) > r.period+1 {
		r.priceHistory = r.priceHistory[1:]
	}
	if len(r.gains) > r.period {
		r.gains = r.gains[1:]
	}
	if len(r.losses) > r.period {
		r.losses = r.losses[1:]
	}

	// Need enough data for RSI calculation
	if len(r.gains) < r.period {
		return signal, nil
	}

	// Calculate average gain and loss
	avgGain := r.calculateAverage(r.gains)
	avgLoss := r.calculateAverage(r.losses)

	// Calculate RSI
	var rsi float64
	if avgLoss == 0 {
		rsi = 100.0
	} else {
		rs := avgGain / avgLoss
		rsi = 100.0 - (100.0 / (1.0 + rs))
	}

	signal.Metadata["rsi"] = rsi
	signal.Metadata["overbought_level"] = r.overboughtLevel
	signal.Metadata["oversold_level"] = r.oversoldLevel

	// Generate signals based on RSI levels and crossovers
	if r.lastRSI > r.oversoldLevel && rsi <= r.oversoldLevel {
		// RSI entered oversold territory - potential buy signal
		signal.Type = SignalBuy
		signal.Strength = (r.oversoldLevel - rsi) / r.oversoldLevel
	} else if r.lastRSI < r.overboughtLevel && rsi >= r.overboughtLevel {
		// RSI entered overbought territory - potential sell signal
		signal.Type = SignalSell
		signal.Strength = (rsi - r.overboughtLevel) / (100.0 - r.overboughtLevel)
	}

	// Ensure strength is between 0 and 1
	if signal.Strength < 0 {
		signal.Strength = 0
	} else if signal.Strength > 1 {
		signal.Strength = 1
	}

	r.lastRSI = rsi

	return signal, nil
}

func (r *RSISignal) calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	
	sum := 0.0
	for _, val := range values {
		sum += val
	}
	return sum / float64(len(values))
}

func (r *RSISignal) GetDefaultParameters() map[string]interface{} {
	return map[string]interface{}{
		"period":           14,
		"overbought_level": 70.0,
		"oversold_level":   30.0,
	}
}

func (r *RSISignal) ValidateParameters(params map[string]interface{}) error {
	period, err := r.GetIntParameter("period", 14)
	if err != nil {
		return err
	}
	
	if period <= 0 {
		return fmt.Errorf("period must be greater than 0")
	}
	
	overboughtLevel, err := r.GetFloatParameter("overbought_level", 70.0)
	if err != nil {
		return err
	}
	
	oversoldLevel, err := r.GetFloatParameter("oversold_level", 30.0)
	if err != nil {
		return err
	}
	
	if overboughtLevel <= oversoldLevel {
		return fmt.Errorf("overbought_level must be greater than oversold_level")
	}
	
	if overboughtLevel > 100 || overboughtLevel < 0 {
		return fmt.Errorf("overbought_level must be between 0 and 100")
	}
	
	if oversoldLevel > 100 || oversoldLevel < 0 {
		return fmt.Errorf("oversold_level must be between 0 and 100")
	}
	
	return nil
}

func (r *RSISignal) Reset() error {
	r.gains = make([]float64, 0, r.period)
	r.losses = make([]float64, 0, r.period)
	r.priceHistory = make([]float64, 0, r.period+1)
	r.lastRSI = 50.0
	return nil
}