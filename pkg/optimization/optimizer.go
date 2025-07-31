package optimization

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/tradestax/traedor/pkg/storage"
)

type Optimizer struct {
	storage storage.IStorage
}

func NewOptimizer(store storage.IStorage) *Optimizer {
	return &Optimizer{
		storage: store,
	}
}

// GenerateParameterCombinations creates all possible parameter combinations
func (o *Optimizer) GenerateParameterCombinations(config storage.OptimizationConfig) ([]map[string]interface{}, error) {
	if len(config.ParameterRanges) == 0 {
		return []map[string]interface{}{{}}, nil
	}

	// Generate all possible values for each parameter
	parameterValues := make([][]parameterValue, len(config.ParameterRanges))
	
	for i, paramRange := range config.ParameterRanges {
		values, err := o.generateParameterValues(paramRange)
		if err != nil {
			return nil, fmt.Errorf("failed to generate values for parameter %s: %w", paramRange.ParameterPath, err)
		}
		parameterValues[i] = values
	}

	// Generate cartesian product of all parameter values
	combinations := o.cartesianProduct(parameterValues)
	
	// Convert to the expected format
	result := make([]map[string]interface{}, len(combinations))
	for i, combo := range combinations {
		paramMap := make(map[string]interface{})
		for j, value := range combo {
			paramMap[config.ParameterRanges[j].ParameterPath] = value.Value
		}
		result[i] = paramMap
	}

	// Shuffle if random order is requested
	if config.RandomOrder {
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(result), func(i, j int) {
			result[i], result[j] = result[j], result[i]
		})
	}

	return result, nil
}

type parameterValue struct {
	Value interface{}
	Path  string
}

func (o *Optimizer) generateParameterValues(paramRange storage.OptimizationParameterRange) ([]parameterValue, error) {
	var values []parameterValue

	switch paramRange.ParameterType {
	case "int":
		lower, ok := paramRange.LowerBound.(float64)
		if !ok {
			return nil, fmt.Errorf("lower bound must be a number for int parameter")
		}
		upper, ok := paramRange.UpperBound.(float64)
		if !ok {
			return nil, fmt.Errorf("upper bound must be a number for int parameter")
		}
		step, ok := paramRange.Step.(float64)
		if !ok {
			return nil, fmt.Errorf("step must be a number for int parameter")
		}

		for val := int(lower); val <= int(upper); val += int(step) {
			values = append(values, parameterValue{
				Value: val,
				Path:  paramRange.ParameterPath,
			})
		}

	case "float":
		lower, ok := paramRange.LowerBound.(float64)
		if !ok {
			return nil, fmt.Errorf("lower bound must be a number for float parameter")
		}
		upper, ok := paramRange.UpperBound.(float64)
		if !ok {
			return nil, fmt.Errorf("upper bound must be a number for float parameter")
		}
		step, ok := paramRange.Step.(float64)
		if !ok {
			return nil, fmt.Errorf("step must be a number for float parameter")
		}

		for val := lower; val <= upper; val += step {
			// Round to avoid floating point precision issues
			rounded := math.Round(val*1000) / 1000
			values = append(values, parameterValue{
				Value: rounded,
				Path:  paramRange.ParameterPath,
			})
		}

	case "string":
		// For string parameters, expect discrete values in bounds
		bounds, ok := paramRange.LowerBound.([]interface{})
		if !ok {
			return nil, fmt.Errorf("for string parameters, lower_bound should contain array of possible values")
		}

		for _, val := range bounds {
			str, ok := val.(string)
			if !ok {
				return nil, fmt.Errorf("all values in string parameter array must be strings")
			}
			values = append(values, parameterValue{
				Value: str,
				Path:  paramRange.ParameterPath,
			})
		}

	default:
		return nil, fmt.Errorf("unsupported parameter type: %s", paramRange.ParameterType)
	}

	return values, nil
}

func (o *Optimizer) cartesianProduct(input [][]parameterValue) [][]parameterValue {
	if len(input) == 0 {
		return [][]parameterValue{{}}
	}

	result := [][]parameterValue{{}}
	
	for _, values := range input {
		var newResult [][]parameterValue
		for _, existing := range result {
			for _, value := range values {
				newCombo := make([]parameterValue, len(existing)+1)
				copy(newCombo, existing)
				newCombo[len(existing)] = value
				newResult = append(newResult, newCombo)
			}
		}
		result = newResult
	}

	return result
}

// ApplyParametersToRunConfig applies the parameter combination to the base run configuration
func (o *Optimizer) ApplyParametersToRunConfig(baseConfig storage.RunConfig, parameters map[string]interface{}) (storage.RunConfig, error) {
	// Deep copy the config
	configBytes, err := json.Marshal(baseConfig)
	if err != nil {
		return storage.RunConfig{}, fmt.Errorf("failed to marshal base config: %w", err)
	}
	
	var modifiedConfig storage.RunConfig
	if err := json.Unmarshal(configBytes, &modifiedConfig); err != nil {
		return storage.RunConfig{}, fmt.Errorf("failed to unmarshal base config: %w", err)
	}

	// Apply each parameter
	for path, value := range parameters {
		if err := o.setNestedValue(&modifiedConfig, path, value); err != nil {
			return storage.RunConfig{}, fmt.Errorf("failed to set parameter %s: %w", path, err)
		}
	}

	return modifiedConfig, nil
}

func (o *Optimizer) setNestedValue(obj interface{}, path string, value interface{}) error {
	parts := strings.Split(path, ".")
	
	// Navigate to the parent of the target field
	current := reflect.ValueOf(obj)
	if current.Kind() != reflect.Ptr {
		return fmt.Errorf("root object must be a pointer")
	}
	current = current.Elem()

	for i, part := range parts[:len(parts)-1] {
		if current.Kind() == reflect.Slice {
			// Handle array indices
			index, err := strconv.Atoi(part)
			if err != nil {
				return fmt.Errorf("invalid array index %s at path segment %d", part, i)
			}
			if index >= current.Len() {
				return fmt.Errorf("array index %d out of bounds (length %d) at path segment %d", index, current.Len(), i)
			}
			current = current.Index(index)
			if current.Kind() == reflect.Ptr {
				current = current.Elem()
			}
		} else if current.Kind() == reflect.Struct {
			// Handle struct fields
			fieldName := o.jsonTagToFieldName(current.Type(), part)
			field := current.FieldByName(fieldName)
			if !field.IsValid() {
				return fmt.Errorf("field %s not found at path segment %d", part, i)
			}
			current = field
			if current.Kind() == reflect.Ptr && current.IsNil() {
				// Initialize nil pointer
				current.Set(reflect.New(current.Type().Elem()))
			}
			if current.Kind() == reflect.Ptr {
				current = current.Elem()
			}
		} else if current.Kind() == reflect.Map {
			// Handle map navigation
			mapKey := reflect.ValueOf(part)
			mapValue := current.MapIndex(mapKey)
			if !mapValue.IsValid() {
				// Key doesn't exist - this is ok for intermediate navigation
				// We'll create it when we set the final value
				return fmt.Errorf("map key %s not found at path segment %d", part, i)
			}
			current = mapValue
		} else {
			return fmt.Errorf("cannot navigate path %s at segment %d: not a struct, slice, or map", path, i)
		}
	}

	// Set the final value
	finalPart := parts[len(parts)-1]
	if current.Kind() == reflect.Slice {
		index, err := strconv.Atoi(finalPart)
		if err != nil {
			return fmt.Errorf("invalid array index %s", finalPart)
		}
		if index >= current.Len() {
			return fmt.Errorf("array index %d out of bounds (length %d)", index, current.Len())
		}
		targetField := current.Index(index)
		if targetField.Kind() == reflect.Ptr {
			targetField = targetField.Elem()
		}
		return o.setReflectValue(targetField, value)
	} else if current.Kind() == reflect.Struct {
		fieldName := o.jsonTagToFieldName(current.Type(), finalPart)
		field := current.FieldByName(fieldName)
		if !field.IsValid() {
			return fmt.Errorf("field %s not found", finalPart)
		}
		return o.setReflectValue(field, value)
	} else if current.Kind() == reflect.Map {
		// Handle map[string]interface{} case
		mapKey := reflect.ValueOf(finalPart)
		mapValue := reflect.ValueOf(value)
		current.SetMapIndex(mapKey, mapValue)
		return nil
	}

	return fmt.Errorf("cannot set value at path %s", path)
}

func (o *Optimizer) jsonTagToFieldName(structType reflect.Type, jsonTag string) string {
	for i := 0; i < structType.NumField(); i++ {
		field := structType.Field(i)
		tag := field.Tag.Get("json")
		tagName := strings.Split(tag, ",")[0]
		if tagName == jsonTag {
			return field.Name
		}
	}
	// Fallback: try to match field name directly (with first letter uppercase)
	if len(jsonTag) > 0 {
		return strings.ToUpper(jsonTag[:1]) + jsonTag[1:]
	}
	return jsonTag
}

func (o *Optimizer) setReflectValue(field reflect.Value, value interface{}) error {
	if !field.CanSet() {
		return fmt.Errorf("field cannot be set")
	}

	valueReflect := reflect.ValueOf(value)
	fieldType := field.Type()

	// Handle type conversion
	if valueReflect.Type().ConvertibleTo(fieldType) {
		field.Set(valueReflect.Convert(fieldType))
		return nil
	}

	// Handle interface{} fields
	if fieldType.Kind() == reflect.Interface {
		field.Set(valueReflect)
		return nil
	}

	return fmt.Errorf("cannot convert %v (%T) to %v", value, value, fieldType)
}

// CalculateOptimizationScore calculates the optimization score based on the metric
func (o *Optimizer) CalculateOptimizationScore(metrics *storage.PerformanceMetrics, optimizationMetric string) float64 {
	if metrics == nil {
		return math.Inf(-1) // Worst possible score for failed runs
	}

	switch optimizationMetric {
	case "cumulative_return":
		return metrics.ReturnPercentage
	case "total_profit":
		return metrics.TotalProfit
	case "sharpe_ratio":
		return metrics.SharpeRatio
	case "profit_factor":
		return metrics.ProfitFactor
	case "win_rate":
		return metrics.WinRate
	case "max_drawdown":
		return -metrics.MaxDrawdownPercent // Negative because lower drawdown is better
	default:
		// Default to cumulative return
		return metrics.ReturnPercentage
	}
}