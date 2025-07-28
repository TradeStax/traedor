package signals

import (
	"fmt"
	"sync"

	"github.com/tradestax/traedor/pkg/storage"
)

type SignalManager struct {
	generators map[string]ISignalGenerator
	storage    storage.IStorage
	mu         sync.RWMutex
}

func NewSignalManager(store storage.IStorage) *SignalManager {
	return &SignalManager{
		generators: make(map[string]ISignalGenerator),
		storage:    store,
	}
}

func (sm *SignalManager) LoadSignalDefinitions() error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	definitions, err := sm.storage.GetSignalDefinitions()
	if err != nil {
		return fmt.Errorf("failed to load signal definitions: %w", err)
	}

	for _, def := range definitions {
		if !def.Active {
			continue
		}

		generator, exists := GetSignalGenerator(def.Type)
		if !exists {
			return fmt.Errorf("signal generator '%s' not found for definition '%s'", def.Type, def.Name)
		}

		if err := generator.Initialize(def.Parameters); err != nil {
			return fmt.Errorf("failed to initialize signal '%s': %w", def.Name, err)
		}

		sm.generators[def.ID] = generator
	}

	return nil
}

func (sm *SignalManager) GetGenerator(signalID string) (ISignalGenerator, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	generator, exists := sm.generators[signalID]
	return generator, exists
}

func (sm *SignalManager) AddSignal(def storage.SignalDefinition) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate the signal definition
	generator, exists := GetSignalGenerator(def.Type)
	if !exists {
		return fmt.Errorf("signal generator '%s' not found", def.Type)
	}

	if err := generator.ValidateParameters(def.Parameters); err != nil {
		return fmt.Errorf("invalid parameters for signal '%s': %w", def.Name, err)
	}

	// Save to storage
	if err := sm.storage.CreateSignalDefinition(def); err != nil {
		return fmt.Errorf("failed to save signal definition: %w", err)
	}

	// Initialize and add to active generators if active
	if def.Active {
		if err := generator.Initialize(def.Parameters); err != nil {
			return fmt.Errorf("failed to initialize signal '%s': %w", def.Name, err)
		}
		sm.generators[def.ID] = generator
	}

	return nil
}

func (sm *SignalManager) UpdateSignal(id string, def storage.SignalDefinition) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate the signal definition
	generator, exists := GetSignalGenerator(def.Type)
	if !exists {
		return fmt.Errorf("signal generator '%s' not found", def.Type)
	}

	if err := generator.ValidateParameters(def.Parameters); err != nil {
		return fmt.Errorf("invalid parameters for signal '%s': %w", def.Name, err)
	}

	// Update in storage
	if err := sm.storage.UpdateSignalDefinition(id, def); err != nil {
		return fmt.Errorf("failed to update signal definition: %w", err)
	}

	// Remove from active generators if exists
	delete(sm.generators, id)

	// Initialize and add to active generators if active
	if def.Active {
		if err := generator.Initialize(def.Parameters); err != nil {
			return fmt.Errorf("failed to initialize signal '%s': %w", def.Name, err)
		}
		sm.generators[id] = generator
	}

	return nil
}

func (sm *SignalManager) RemoveSignal(id string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Remove from storage
	if err := sm.storage.DeleteSignalDefinition(id); err != nil {
		return fmt.Errorf("failed to delete signal definition: %w", err)
	}

	// Remove from active generators
	delete(sm.generators, id)

	return nil
}

func (sm *SignalManager) ListActiveGenerators() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	ids := make([]string, 0, len(sm.generators))
	for id := range sm.generators {
		ids = append(ids, id)
	}
	return ids
}

func (sm *SignalManager) GetAvailableSignalTypes() []map[string]interface{} {
	availableTypes := GetAvailableSignalGenerators()
	result := make([]map[string]interface{}, len(availableTypes))

	for i, signalType := range availableTypes {
		generator, _ := GetSignalGenerator(signalType)
		result[i] = map[string]interface{}{
			"name":        signalType,
			"description": generator.GetDescription(),
			"parameters":  generator.GetDefaultParameters(),
		}
	}

	return result
}

type SignalWorkflow struct {
	manager    *SignalManager
	activeRuns map[string]*RunSignalContext
	mu         sync.RWMutex
}

type RunSignalContext struct {
	RunID      string
	Generators map[string]ISignalGenerator
}

func NewSignalWorkflow(store storage.IStorage) *SignalWorkflow {
	return &SignalWorkflow{
		manager:    NewSignalManager(store),
		activeRuns: make(map[string]*RunSignalContext),
	}
}

func (sw *SignalWorkflow) InitializeRun(runID string, signalIDs []string) error {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	// Load signal definitions if not loaded
	if err := sw.manager.LoadSignalDefinitions(); err != nil {
		return fmt.Errorf("failed to load signal definitions: %w", err)
	}

	runContext := &RunSignalContext{
		RunID:      runID,
		Generators: make(map[string]ISignalGenerator),
	}

	// Initialize generators for this run
	for _, signalID := range signalIDs {
		_, exists := sw.manager.GetGenerator(signalID)
		if !exists {
			return fmt.Errorf("signal generator '%s' not found", signalID)
		}

		// Create a new instance for this run to avoid state conflicts
		definitions, err := sw.manager.storage.GetSignalDefinitions()
		if err != nil {
			return fmt.Errorf("failed to get signal definitions: %w", err)
		}

		var def storage.SignalDefinition
		found := false
		for _, d := range definitions {
			if d.ID == signalID {
				def = d
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("signal definition '%s' not found", signalID)
		}

		// Check if this signal has aggregation parameters
		var generator ISignalGenerator
		if aggregationInterval, hasAggregation := def.Parameters["aggregation_interval"]; hasAggregation {
			// Create aggregated signal
			intervalMinutes, ok := aggregationInterval.(float64)
			if !ok {
				return fmt.Errorf("invalid aggregation_interval type for signal '%s'", def.Name)
			}

			baseGenerator, exists := GetSignalGenerator(def.Type)
			if !exists {
				return fmt.Errorf("signal generator '%s' not found", def.Type)
			}

			// Create aggregated wrapper
			generator = NewAggregatedSignal(baseGenerator, int(intervalMinutes))
		} else {
			// Create regular signal
			var exists bool
			generator, exists = GetSignalGenerator(def.Type)
			if !exists {
				return fmt.Errorf("signal generator '%s' not found", def.Type)
			}
		}

		if err := generator.Initialize(def.Parameters); err != nil {
			return fmt.Errorf("failed to initialize signal '%s': %w", def.Name, err)
		}

		runContext.Generators[signalID] = generator
	}

	sw.activeRuns[runID] = runContext
	return nil
}

func (sw *SignalWorkflow) GetRunGenerators(runID string) (map[string]ISignalGenerator, bool) {
	sw.mu.RLock()
	defer sw.mu.RUnlock()

	runContext, exists := sw.activeRuns[runID]
	if !exists {
		return nil, false
	}

	return runContext.Generators, true
}

func (sw *SignalWorkflow) CleanupRun(runID string) {
	sw.mu.Lock()
	defer sw.mu.Unlock()

	delete(sw.activeRuns, runID)
}

func (sw *SignalWorkflow) GetManager() *SignalManager {
	return sw.manager
}