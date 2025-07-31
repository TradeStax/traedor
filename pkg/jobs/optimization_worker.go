package jobs

import (
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/tradestax/traedor/internal/config"
	"github.com/tradestax/traedor/pkg/optimization"
	"github.com/tradestax/traedor/pkg/storage"
)

type OptimizationWorker struct {
	id       string
	config   *config.Config
	storage  storage.IStorage
	optimizer *optimization.Optimizer
	
	// Current optimization tracking
	currentOptimization   *storage.Optimization
	currentOptimizationMu sync.RWMutex
	
	// Worker state
	running bool
	stopCh  chan struct{}
	doneCh  chan struct{}
}

func NewOptimizationWorker(id string, cfg *config.Config, store storage.IStorage) *OptimizationWorker {
	return &OptimizationWorker{
		id:        id,
		config:    cfg,
		storage:   store,
		optimizer: optimization.NewOptimizer(store),
		stopCh:    make(chan struct{}),
		doneCh:    make(chan struct{}),
	}
}

func (w *OptimizationWorker) Start() {
	w.running = true
	go w.run()
}

func (w *OptimizationWorker) Stop() {
	if !w.running {
		return
	}
	
	close(w.stopCh)
	<-w.doneCh
	w.running = false
}

func (w *OptimizationWorker) run() {
	defer close(w.doneCh)
	
	log.Printf("Optimization worker %s started", w.id)
	
	for {
		select {
		case <-w.stopCh:
			log.Printf("Optimization worker %s stopping", w.id)
			
			// Release any claimed optimization
			w.currentOptimizationMu.Lock()
			if w.currentOptimization != nil {
				if err := w.storage.ReleaseOptimizationClaim(w.currentOptimization.ID); err != nil {
					log.Printf("Failed to release optimization claim on shutdown: %v", err)
				}
				w.currentOptimization = nil
			}
			w.currentOptimizationMu.Unlock()
			
			return
			
		default:
			// Try to claim a queued optimization
			opt, err := w.storage.ClaimNextQueuedOptimization(w.id)
			if err != nil {
				log.Printf("Error claiming optimization: %v", err)
				time.Sleep(5 * time.Second)
				continue
			}
			
			if opt == nil {
				// No optimization available, wait and try again
				time.Sleep(2 * time.Second)
				continue
			}
			
			// Process the optimization
			w.currentOptimizationMu.Lock()
			w.currentOptimization = opt
			w.currentOptimizationMu.Unlock()
			
			if err := w.processOptimization(opt); err != nil {
				log.Printf("Error processing optimization %s: %v", opt.ID, err)
				
				// Mark optimization as failed
				if updateErr := w.storage.UpdateOptimizationStatus(opt.ID, storage.OptimizationStatusFailed, 0, fmt.Sprintf("Processing failed: %v", err)); updateErr != nil {
					log.Printf("Failed to update optimization status to failed: %v", updateErr)
				}
			}
			
			w.currentOptimizationMu.Lock()
			w.currentOptimization = nil
			w.currentOptimizationMu.Unlock()
		}
	}
}

func (w *OptimizationWorker) processOptimization(opt *storage.Optimization) error {
	log.Printf("Processing optimization %s with %d parameter combinations", opt.ID, opt.TotalPermutations)
	
	// Update status to running
	if err := w.storage.UpdateOptimizationStatus(opt.ID, storage.OptimizationStatusRunning, 0, "Starting optimization"); err != nil {
		return fmt.Errorf("failed to update optimization status: %w", err)
	}
	
	// Get or generate parameter sequence
	parameterSequence := opt.ParameterSequence
	if len(parameterSequence) == 0 {
		log.Printf("Generating parameter combinations for optimization %s", opt.ID)
		var err error
		parameterSequence, err = w.optimizer.GenerateParameterCombinations(opt.Config)
		if err != nil {
			return fmt.Errorf("failed to generate parameter combinations: %w", err)
		}
		
		// Update with the parameter sequence
		if err := w.storage.UpdateOptimizationSequence(opt.ID, len(parameterSequence), parameterSequence); err != nil {
			log.Printf("Warning: Failed to update optimization sequence: %v", err)
		}
	}
	
	// Track completed runs to ensure resumability
	existingRuns, err := w.storage.GetOptimizationRuns(opt.ID)
	if err != nil {
		log.Printf("Warning: Failed to get existing optimization runs: %v", err)
		existingRuns = []*storage.OptimizationRun{}
	}
	
	// Create a map to track which parameter indices have been completed
	completedIndices := make(map[int]bool)
	completedCount := 0
	
	// Also check for existing backtests with the same configurations
	// This handles the case where backtests were run outside of this optimization
	for paramIndex, parameters := range parameterSequence {
		// Skip if we already have an optimization run for this parameter index
		if w.findOptimizationRunForParameterIndex(existingRuns, paramIndex) != nil {
			continue
		}
		
		// Apply parameters to create the run config
		modifiedConfig, err := w.optimizer.ApplyParametersToRunConfig(opt.Config.BaseRunConfig, parameters)
		if err != nil {
			log.Printf("Warning: Failed to apply parameters for index %d: %v", paramIndex, err)
			continue
		}
		
		// Check if we already have runs with this exact config
		existingBacktests, err := w.storage.GetRunsByConfig(modifiedConfig)
		if err != nil {
			log.Printf("Warning: Failed to check for existing backtests for parameter index %d: %v", paramIndex, err)
			continue
		}
		
		// Find the first completed or failed backtest with this config
		var selectedBacktest *storage.Run
		for _, backtest := range existingBacktests {
			if backtest.Status == storage.RunStatusCompleted || backtest.Status == storage.RunStatusFailed {
				selectedBacktest = backtest
				break
			}
		}
		
		// If we found a completed/failed backtest, create an optimization run record for it
		if selectedBacktest != nil {
			completedIndices[paramIndex] = true
			completedCount++
			
			// Create new optimization run record linked to existing backtest
			newOptRun, err := w.storage.CreateOptimizationRun(opt.ID, modifiedConfig, paramIndex)
			if err != nil {
				log.Printf("Failed to create optimization run record for existing backtest: %v", err)
				continue
			}
			
			// Update the optimization run to link to the existing backtest
			status := storage.RunStatusCompleted
			if selectedBacktest.Status == storage.RunStatusFailed {
				status = storage.RunStatusFailed
			}
			if err := w.storage.UpdateOptimizationRunStatus(newOptRun.ID, status, selectedBacktest.ID, selectedBacktest.PerformanceMetrics); err != nil {
				log.Printf("Failed to update optimization run status for existing backtest: %v", err)
			}
		}
	}
	
	// Also check optimization runs that we already know about
	for _, run := range existingRuns {
		if run.Status == storage.RunStatusCompleted || run.Status == storage.RunStatusFailed {
			if !completedIndices[run.ParameterIndex] {
				completedIndices[run.ParameterIndex] = true
				completedCount++
			}
		}
	}
	
	log.Printf("Resuming optimization %s: %d/%d runs already completed (including existing backtests)", opt.ID, completedCount, len(parameterSequence))
	
	// Process each parameter combination that hasn't been completed
	workerCount := 2 // Get this from config or environment
	if w.config != nil && w.config.Workers.Count > 0 {
		workerCount = w.config.Workers.Count
	}
	
	// Create a semaphore to limit concurrent backtests
	semaphore := make(chan struct{}, workerCount)
	var wg sync.WaitGroup
	
	for paramIndex, parameters := range parameterSequence {
		// Skip if already completed
		if completedIndices[paramIndex] {
			// Still need to update progress when skipping completed runs
			progress := float64(completedCount) / float64(len(parameterSequence)) * 100
			statusMessage := fmt.Sprintf("Processing: %d/%d completed (skipped already completed run)", completedCount, len(parameterSequence))
			if updateErr := w.storage.UpdateOptimizationStatus(opt.ID, storage.OptimizationStatusRunning, progress, statusMessage); updateErr != nil {
				log.Printf("Failed to update optimization progress: %v", updateErr)
			}
			continue
		}
		
		select {
		case <-w.stopCh:
			log.Printf("Optimization worker %s stopping, abandoning optimization %s", w.id, opt.ID)
			return nil
		default:
		}
		
		// Check if optimization has been paused
		currentOpt, err := w.storage.GetOptimization(opt.ID)
		if err != nil {
			log.Printf("Failed to check optimization status: %v", err)
		} else if currentOpt.Status == storage.OptimizationStatusPaused {
			log.Printf("Optimization %s has been paused, stopping processing", opt.ID)
			return nil
		}
		
		wg.Add(1)
		semaphore <- struct{}{} // Acquire semaphore
		
		go func(index int, params map[string]interface{}) {
			defer wg.Done()
			defer func() { <-semaphore }() // Release semaphore
			
			if err := w.processParameterCombination(opt, index, params); err != nil {
				log.Printf("Failed to process parameter combination %d for optimization %s: %v", index, opt.ID, err)
			}
			
			// Update progress
			completedCount++
			progress := float64(completedCount) / float64(len(parameterSequence)) * 100
			
			statusMessage := fmt.Sprintf("Processing: %d/%d completed", completedCount, len(parameterSequence))
			if updateErr := w.storage.UpdateOptimizationStatus(opt.ID, storage.OptimizationStatusRunning, progress, statusMessage); updateErr != nil {
				log.Printf("Failed to update optimization progress: %v", updateErr)
			}
			
		}(paramIndex, parameters)
	}
	
	// Wait for all parameter combinations to complete
	wg.Wait()
	
	log.Printf("All parameter combinations completed for optimization %s", opt.ID)
	
	// Clean up any duplicate results
	if duplicatesRemoved, err := w.storage.CleanupDuplicateOptimizationRunResults(opt.ID); err != nil {
		log.Printf("Warning: Failed to cleanup duplicate optimization run results: %v", err)
	} else if duplicatesRemoved > 0 {
		log.Printf("Cleaned up %d duplicate optimization run results for optimization %s", duplicatesRemoved, opt.ID)
	}
	
	// Calculate final results
	if err := w.calculateOptimizationResults(opt); err != nil {
		return fmt.Errorf("failed to calculate optimization results: %w", err)
	}
	
	// Mark optimization as completed
	if err := w.storage.UpdateOptimizationStatus(opt.ID, storage.OptimizationStatusCompleted, 100, "Optimization completed successfully"); err != nil {
		return fmt.Errorf("failed to update optimization status to completed: %w", err)
	}
	
	log.Printf("Optimization %s completed successfully", opt.ID)
	return nil
}

func (w *OptimizationWorker) processParameterCombination(opt *storage.Optimization, paramIndex int, parameters map[string]interface{}) error {
	// Create optimization run record
	modifiedConfig, err := w.optimizer.ApplyParametersToRunConfig(opt.Config.BaseRunConfig, parameters)
	if err != nil {
		return fmt.Errorf("failed to apply parameters to run config: %w", err)
	}
	
	optimizationRun, err := w.storage.CreateOptimizationRun(opt.ID, modifiedConfig, paramIndex)
	if err != nil {
		return fmt.Errorf("failed to create optimization run: %w", err)
	}
	
	// Create the actual backtest run
	backtestRun, err := w.storage.CreateRun(modifiedConfig)
	if err != nil {
		return fmt.Errorf("failed to create backtest run: %w", err)
	}
	
	// Update optimization run with backtest run ID
	if err := w.storage.UpdateOptimizationRunStatus(optimizationRun.ID, storage.RunStatusQueued, backtestRun.ID, nil); err != nil {
		return fmt.Errorf("failed to update optimization run status: %w", err)
	}
	
	// Queue the backtest run for processing
	if err := w.storage.UpdateRunStatus(backtestRun.ID, storage.RunStatusQueued, nil); err != nil {
		return fmt.Errorf("failed to queue backtest run: %w", err)
	}
	
	// Wait for backtest to complete (with timeout)
	timeout := 30 * time.Minute // Configurable timeout
	start := time.Now()
	
	for {
		if time.Since(start) > timeout {
			return fmt.Errorf("backtest run %s timed out", backtestRun.ID)
		}
		
		// Check if we should stop
		select {
		case <-w.stopCh:
			return fmt.Errorf("worker stopping, abandoning backtest %s", backtestRun.ID)
		default:
		}
		
		// Check backtest status
		run, err := w.storage.GetRun(backtestRun.ID)
		if err != nil {
			log.Printf("Failed to get backtest run status: %v", err)
			time.Sleep(5 * time.Second)
			continue
		}
		
		switch run.Status {
		case storage.RunStatusCompleted:
			// Backtest completed successfully
			if err := w.storage.UpdateOptimizationRunStatus(optimizationRun.ID, storage.RunStatusCompleted, backtestRun.ID, run.PerformanceMetrics); err != nil {
				return fmt.Errorf("failed to update optimization run status to completed: %w", err)
			}
			return nil
			
		case storage.RunStatusFailed:
			// Backtest failed
			if err := w.storage.UpdateOptimizationRunStatus(optimizationRun.ID, storage.RunStatusFailed, backtestRun.ID, nil); err != nil {
				return fmt.Errorf("failed to update optimization run status to failed: %w", err)
			}
			return fmt.Errorf("backtest run %s failed: %s", backtestRun.ID, run.StatusMessage)
			
		case storage.RunStatusCancelled:
			// Backtest was cancelled
			if err := w.storage.UpdateOptimizationRunStatus(optimizationRun.ID, storage.RunStatusCancelled, backtestRun.ID, nil); err != nil {
				return fmt.Errorf("failed to update optimization run status to cancelled: %w", err)
			}
			return fmt.Errorf("backtest run %s was cancelled", backtestRun.ID)
			
		default:
			// Still running, wait and check again
			time.Sleep(5 * time.Second)
		}
	}
}

func (w *OptimizationWorker) findOptimizationRunForParameterIndex(runs []*storage.OptimizationRun, parameterIndex int) *storage.OptimizationRun {
	for _, run := range runs {
		if run.ParameterIndex == parameterIndex {
			return run
		}
	}
	return nil
}

func (w *OptimizationWorker) calculateOptimizationResults(opt *storage.Optimization) error {
	// Get all completed optimization run results
	results, err := w.storage.GetOptimizationRunResults(opt.ID)
	if err != nil {
		return fmt.Errorf("failed to get optimization run results: %w", err)
	}
	
	if len(results) == 0 {
		return fmt.Errorf("no completed optimization runs found")
	}
	
	// Sort results by optimization score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].OptimizationScore > results[j].OptimizationScore
	})
	
	// Calculate statistics
	var totalReturns float64
	var validReturns []float64
	successfulBacktests := 0
	
	for _, result := range results {
		if result.PerformanceMetrics != nil && !math.IsInf(result.OptimizationScore, -1) {
			totalReturns += result.PerformanceMetrics.ReturnPercentage
			validReturns = append(validReturns, result.PerformanceMetrics.ReturnPercentage)
			successfulBacktests++
		}
	}
	
	averageReturn := 0.0
	medianReturn := 0.0
	
	if successfulBacktests > 0 {
		averageReturn = totalReturns / float64(successfulBacktests)
		
		// Calculate median
		sort.Float64s(validReturns)
		if len(validReturns)%2 == 0 {
			medianReturn = (validReturns[len(validReturns)/2-1] + validReturns[len(validReturns)/2]) / 2
		} else {
			medianReturn = validReturns[len(validReturns)/2]
		}
	}
	
	// Calculate completion time - use CompletedAt if available (for cancelled optimizations), otherwise use current time
	var completionTime time.Duration
	if opt.CompletedAt != nil {
		completionTime = opt.CompletedAt.Sub(opt.StartedAt)
	} else {
		completionTime = time.Since(opt.StartedAt)
	}

	// Create optimization results
	optimizationResults := &storage.OptimizationResults{
		AverageReturn:       averageReturn,
		MedianReturn:        medianReturn,
		TotalBacktests:      len(results),
		SuccessfulBacktests: successfulBacktests,
		FailedBacktests:     len(results) - successfulBacktests,
		CompletionTime:      completionTime,
	}
	
	if len(results) > 0 {
		optimizationResults.BestResult = results[0]
		optimizationResults.BestParameters = results[0].Parameters
		
		if len(results) > 1 {
			optimizationResults.WorstResult = results[len(results)-1]
		}
	}
	
	// Store results
	if err := w.storage.UpdateOptimizationResults(opt.ID, optimizationResults); err != nil {
		return fmt.Errorf("failed to update optimization results: %w", err)
	}
	
	return nil
}