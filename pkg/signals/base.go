package signals

import (
	"fmt"
)

type BaseSignalGenerator struct {
	name        string
	description string
	parameters  map[string]interface{}
}

func NewBaseSignalGenerator(name, description string) BaseSignalGenerator {
	return BaseSignalGenerator{
		name:        name,
		description: description,
		parameters:  make(map[string]interface{}),
	}
}

func (b *BaseSignalGenerator) GetName() string {
	return b.name
}

func (b *BaseSignalGenerator) GetDescription() string {
	return b.description
}

func (b *BaseSignalGenerator) Initialize(params map[string]interface{}) error {
	if err := b.ValidateParameters(params); err != nil {
		return err
	}
	b.parameters = params
	return nil
}

func (b *BaseSignalGenerator) ValidateParameters(params map[string]interface{}) error {
	// Base implementation - can be overridden by specific signal generators
	return nil
}

func (b *BaseSignalGenerator) Reset() error {
	// Base implementation - can be overridden by specific signal generators
	return nil
}

func (b *BaseSignalGenerator) GetParameter(key string, defaultValue interface{}) interface{} {
	if val, ok := b.parameters[key]; ok {
		return val
	}
	return defaultValue
}

func (b *BaseSignalGenerator) GetFloatParameter(key string, defaultValue float64) (float64, error) {
	val := b.GetParameter(key, defaultValue)
	switch v := val.(type) {
	case float64:
		return v, nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	default:
		return defaultValue, fmt.Errorf("parameter %s is not a number", key)
	}
}

func (b *BaseSignalGenerator) GetIntParameter(key string, defaultValue int) (int, error) {
	val := b.GetParameter(key, defaultValue)
	switch v := val.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		return int(v), nil
	default:
		return defaultValue, fmt.Errorf("parameter %s is not a number", key)
	}
}

func (b *BaseSignalGenerator) GetStringParameter(key string, defaultValue string) (string, error) {
	val := b.GetParameter(key, defaultValue)
	if str, ok := val.(string); ok {
		return str, nil
	}
	return defaultValue, fmt.Errorf("parameter %s is not a string", key)
}