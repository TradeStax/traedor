package signals

import (
	"fmt"
	"reflect"

	"github.com/tradestax/traedor/pkg/storage"
)

type ParameterValidator struct {
	Type        string
	Required    bool
	MinValue    *float64
	MaxValue    *float64
	AllowedValues []interface{}
	DefaultValue interface{}
}

type SignalValidator struct {
	RequiredParameters map[string]ParameterValidator
}

func NewSignalValidator() *SignalValidator {
	return &SignalValidator{
		RequiredParameters: make(map[string]ParameterValidator),
	}
}

func (sv *SignalValidator) AddParameter(name string, validator ParameterValidator) {
	sv.RequiredParameters[name] = validator
}

func (sv *SignalValidator) ValidateSignalDefinition(def storage.SignalDefinition) error {
	// Check if signal type exists
	_, exists := GetSignalGenerator(def.Type)
	if !exists {
		return fmt.Errorf("signal type '%s' not supported", def.Type)
	}

	// Validate signal name
	if def.Name == "" {
		return fmt.Errorf("signal name cannot be empty")
	}

	// Create a temporary generator to validate parameters
	generator, _ := GetSignalGenerator(def.Type)
	if err := generator.ValidateParameters(def.Parameters); err != nil {
		return fmt.Errorf("parameter validation failed: %w", err)
	}

	return nil
}

func (sv *SignalValidator) ValidateParameters(params map[string]interface{}) error {
	// Check required parameters
	for paramName, validator := range sv.RequiredParameters {
		value, exists := params[paramName]
		if !exists {
			if validator.Required {
				return fmt.Errorf("required parameter '%s' is missing", paramName)
			}
			// Set default value if available
			if validator.DefaultValue != nil {
				params[paramName] = validator.DefaultValue
			}
			continue
		}

		// Validate parameter type and constraints
		if err := sv.validateParameter(paramName, value, validator); err != nil {
			return err
		}
	}

	return nil
}

func (sv *SignalValidator) validateParameter(name string, value interface{}, validator ParameterValidator) error {
	// Type validation
	expectedType := validator.Type
	actualType := reflect.TypeOf(value).String()

	// Handle numeric type conversions
	if expectedType == "float64" {
		switch v := value.(type) {
		case int:
			value = float64(v)
		case int64:
			value = float64(v)
		case float32:
			value = float64(v)
		case float64:
			// Already correct type
		default:
			return fmt.Errorf("parameter '%s' must be a number, got %s", name, actualType)
		}
	} else if expectedType == "int" {
		switch v := value.(type) {
		case int:
			// Already correct type
		case int64:
			value = int(v)
		case float64:
			if v != float64(int(v)) {
				return fmt.Errorf("parameter '%s' must be an integer, got %f", name, v)
			}
			value = int(v)
		default:
			return fmt.Errorf("parameter '%s' must be an integer, got %s", name, actualType)
		}
	} else if expectedType == "string" {
		if _, ok := value.(string); !ok {
			return fmt.Errorf("parameter '%s' must be a string, got %s", name, actualType)
		}
	} else if expectedType == "bool" {
		if _, ok := value.(bool); !ok {
			return fmt.Errorf("parameter '%s' must be a boolean, got %s", name, actualType)
		}
	}

	// Range validation for numeric types
	if expectedType == "float64" || expectedType == "int" {
		numValue := 0.0
		if expectedType == "float64" {
			numValue = value.(float64)
		} else {
			numValue = float64(value.(int))
		}

		if validator.MinValue != nil && numValue < *validator.MinValue {
			return fmt.Errorf("parameter '%s' must be >= %f, got %f", name, *validator.MinValue, numValue)
		}

		if validator.MaxValue != nil && numValue > *validator.MaxValue {
			return fmt.Errorf("parameter '%s' must be <= %f, got %f", name, *validator.MaxValue, numValue)
		}
	}

	// Allowed values validation
	if len(validator.AllowedValues) > 0 {
		found := false
		for _, allowedValue := range validator.AllowedValues {
			if reflect.DeepEqual(value, allowedValue) {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("parameter '%s' has invalid value '%v', allowed values: %v", 
				name, value, validator.AllowedValues)
		}
	}

	return nil
}

// GetBuiltinValidators returns validators for built-in signal types
func GetBuiltinValidators() map[string]*SignalValidator {
	validators := make(map[string]*SignalValidator)

	// SMA Crossover validator
	smaValidator := NewSignalValidator()
	minVal := 1.0
	maxVal := 1000.0
	smaValidator.AddParameter("short_period", ParameterValidator{
		Type:         "int",
		Required:     true,
		MinValue:     &minVal,
		MaxValue:     &maxVal,
		DefaultValue: 20,
	})
	smaValidator.AddParameter("long_period", ParameterValidator{
		Type:         "int",
		Required:     true,
		MinValue:     &minVal,
		MaxValue:     &maxVal,
		DefaultValue: 50,
	})
	validators["sma_crossover"] = smaValidator

	// RSI validator
	rsiValidator := NewSignalValidator()
	rsiMinVal := 1.0
	rsiMaxVal := 100.0
	periodMaxVal := 1000.0
	rsiValidator.AddParameter("period", ParameterValidator{
		Type:         "int",
		Required:     true,
		MinValue:     &rsiMinVal,
		MaxValue:     &periodMaxVal,
		DefaultValue: 14,
	})
	rsiValidator.AddParameter("overbought_level", ParameterValidator{
		Type:         "float64",
		Required:     true,
		MinValue:     &rsiMinVal,
		MaxValue:     &rsiMaxVal,
		DefaultValue: 70.0,
	})
	rsiValidator.AddParameter("oversold_level", ParameterValidator{
		Type:         "float64",
		Required:     true,
		MinValue:     &rsiMinVal,
		MaxValue:     &rsiMaxVal,
		DefaultValue: 30.0,
	})
	validators["rsi"] = rsiValidator

	return validators
}

// ValidateSignalCompatibility checks if signals are compatible with each other
func ValidateSignalCompatibility(signalIDs []string, storage storage.IStorage) error {
	if len(signalIDs) <= 1 {
		return nil // Single signal or no signals always compatible
	}

	definitions, err := storage.GetSignalDefinitions()
	if err != nil {
		return fmt.Errorf("failed to get signal definitions: %w", err)
	}

	// Create a map for quick lookup
	defMap := make(map[string]storage.SignalDefinition)
	for _, def := range definitions {
		defMap[def.ID] = def
	}

	// Check each signal exists and is active
	for _, signalID := range signalIDs {
		def, exists := defMap[signalID]
		if !exists {
			return fmt.Errorf("signal '%s' not found", signalID)
		}
		if !def.Active {
			return fmt.Errorf("signal '%s' is not active", signalID)
		}
	}

	// TODO: Add more sophisticated compatibility checks
	// For example, checking for conflicting signal types or parameters

	return nil
}