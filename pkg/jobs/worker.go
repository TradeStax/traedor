package jobs

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tradestax/traedor/internal/config"
	"github.com/tradestax/traedor/pkg/broker"
	"github.com/tradestax/traedor/pkg/broker/profit"
	"github.com/tradestax/traedor/pkg/broker/stop"
	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/storage"
	"github.com/tradestax/traedor/pkg/strategy/types"
	"github.com/tradestax/traedor/pkg/trader"
)

type WorkerPool struct {
	config              *config.Config
	storage             storage.IStorage
	workers             []*Worker
	optimizationWorkers []*OptimizationWorker
	workerCount         int
	ctx                 context.Context
	cancel              context.CancelFunc
	wg                  sync.WaitGroup
}

type Worker struct {
	id      string
	config  *config.Config
	storage storage.IStorage
	ctx     context.Context
}

func NewWorkerPool(cfg *config.Config, store storage.IStorage, workerCount int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &WorkerPool{
		config:              cfg,
		storage:             store,
		workerCount:         workerCount,
		ctx:                 ctx,
		cancel:              cancel,
		workers:             make([]*Worker, workerCount),
		optimizationWorkers: make([]*OptimizationWorker, 1), // Start with 1 optimization worker
	}
	
	// Create backtest workers
	for i := 0; i < workerCount; i++ {
		pool.workers[i] = &Worker{
			id:      fmt.Sprintf("worker-%d", i),
			config:  cfg,
			storage: store,
			ctx:     ctx,
		}
	}
	
	// Create optimization workers
	pool.optimizationWorkers[0] = NewOptimizationWorker("opt-worker-0", cfg, store)
	
	return pool
}

func (wp *WorkerPool) Start() {
	log.Printf("Starting worker pool with %d backtest workers and %d optimization workers", wp.workerCount, len(wp.optimizationWorkers))
	
	// Reset any runs that were stuck in 'running' state from previous shutdown
	if err := wp.storage.ResetStuckRuns(); err != nil {
		log.Printf("Warning: Failed to reset stuck runs on startup: %v", err)
	}
	
	// Reset any optimizations that were stuck in 'running' state from previous shutdown
	if err := wp.storage.ResetStuckOptimizations(); err != nil {
		log.Printf("Warning: Failed to reset stuck optimizations on startup: %v", err)
	}
	
	// Start backtest workers
	for _, worker := range wp.workers {
		wp.wg.Add(1)
		go func(w *Worker) {
			defer wp.wg.Done()
			w.run()
		}(worker)
	}
	
	// Start optimization workers
	for _, optWorker := range wp.optimizationWorkers {
		optWorker.Start()
	}
}

func (wp *WorkerPool) Stop() {
	log.Println("Stopping worker pool...")
	
	// Stop optimization workers first
	for _, optWorker := range wp.optimizationWorkers {
		optWorker.Stop()
	}
	
	// Stop backtest workers
	wp.cancel()
	wp.wg.Wait()
	log.Println("Worker pool stopped")
}

func (w *Worker) run() {
	log.Printf("Worker %s started", w.id)
	
	ticker := time.NewTicker(5 * time.Second) // Poll every 5 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-w.ctx.Done():
			log.Printf("Worker %s stopping", w.id)
			return
		case <-ticker.C:
			if err := w.processNextJob(); err != nil {
				log.Printf("Worker %s error: %v", w.id, err)
			}
		}
	}
}

func (w *Worker) processNextJob() error {
	// Try to claim a queued run
	run, err := w.storage.ClaimNextQueuedRun(w.id)
	if err != nil {
		return fmt.Errorf("failed to claim run: %w", err)
	}
	
	if run == nil {
		// No jobs available
		return nil
	}
	
	log.Printf("Worker %s claimed run %s", w.id, run.ID)
	
	// Ensure we release the claim on exit
	defer func() {
		if err := w.storage.ReleaseRunClaim(run.ID); err != nil {
			log.Printf("Failed to release claim for run %s: %v", run.ID, err)
		}
	}()
	
	// Execute the backtest
	return w.executeBacktest(run)
}

func (w *Worker) executeBacktest(run *storage.Run) error {
	runID := run.ID
	
	// Update status to running
	if err := w.storage.UpdateRunStatus(runID, storage.RunStatusRunning, nil); err != nil {
		return fmt.Errorf("failed to update run status: %w", err)
	}
	
	if err := w.storage.UpdateRunProgress(runID, 5.0, "Initializing backtest..."); err != nil {
		log.Printf("Failed to update progress: %v", err)
	}
	
	// Build trader configuration
	if err := w.storage.UpdateRunProgress(runID, 10.0, "Building trader configuration..."); err != nil {
		log.Printf("Failed to update progress: %v", err)
	}
	traderConfig := w.buildTraderConfig(run.Config)
	
	// Create progress callback
	progressCallback := func(progress float64, message string) {
		if err := w.storage.UpdateRunProgress(runID, progress, message); err != nil {
			log.Printf("Failed to update progress: %v", err)
		}
	}
	
	// Create and run the actual trader
	log.Printf("Starting trader with config: %+v", traderConfig)
	progressCallback(15.0, "Creating trader instance...")
	
	traderInstance := trader.NewTraderWithStorage(traderConfig, w.storage, runID)
	
	progressCallback(20.0, "Initializing market data...")
	
	// Ensure cleanup happens regardless of how the function exits
	defer func() {
		log.Printf("Cleaning up trader resources for run %s", runID)
		traderInstance.Cleanup()
	}()
	
	progressCallback(30.0, "Loading market data...")
	
	// Create progress callback for trader
	traderProgressCallback := func(progress float64, message string) {
		if err := w.storage.UpdateRunProgress(runID, progress, message); err != nil {
			log.Printf("Failed to update trader progress: %v", err)
		}
	}
	
	// Pass progress callback to trader
	traderInstance.SetProgressCallback(traderProgressCallback)
	
	// Run the trader and wait for completion
	log.Printf("Worker: About to call traderInstance.Run() for run %s", runID)
	err := traderInstance.Run()
	log.Printf("Worker: traderInstance.Run() returned for run %s with error: %v", runID, err)
	if err != nil {
		log.Printf("Trader failed: %v", err)
		if updateErr := w.storage.UpdateRunStatus(runID, storage.RunStatusFailed, nil); updateErr != nil {
			log.Printf("Failed to update run status to failed: %v", updateErr)
		}
		return fmt.Errorf("trader execution failed: %w", err)
	}
	
	progressCallback(90.0, "Calculating performance metrics...")
	
	// Run trader summary to calculate and save metrics
	traderInstance.Summary()
	
	progressCallback(100.0, "Completed successfully")
	
	// The trader's Summary() method will handle updating the run status to completed
	// with the real performance metrics, so we don't need to do it here
	
	log.Printf("Worker %s completed run %s successfully", w.id, runID)
	return nil
}

func (w *Worker) buildTraderConfig(runConfig storage.RunConfig) *config.Config {
	// Convert storage.RunConfig to config.Config
	log.Printf("Building trader config - input broker type: '%s'", runConfig.Broker.Type)
	brokerConfig := w.convertBrokerConfig(runConfig.Broker)
	log.Printf("Building trader config - converted broker type: '%s'", brokerConfig.Type)
	datafeedConfigs := w.convertDatafeedConfigs(runConfig.Datafeeds, runConfig.StartTime, runConfig.EndTime, runConfig.Symbol)
	strategyConfigs := w.convertStrategyConfigs(runConfig.Strategies)
	
	// Convert signals to strategies if no explicit strategies are provided
	log.Printf("buildTraderConfig debug: strategyConfigs=%d, Signals=%d, SignalsWithParams=%d", 
		len(strategyConfigs), len(runConfig.Signals), len(runConfig.SignalsWithParams))
	if len(strategyConfigs) == 0 && (len(runConfig.Signals) > 0 || len(runConfig.SignalsWithParams) > 0) {
		// Handle SignalsWithParams (new format for optimization)
		if len(runConfig.SignalsWithParams) > 0 {
			log.Printf("Converting %d SignalsWithParams to strategies", len(runConfig.SignalsWithParams))
			
			// Get signal definitions to extract the proper signal types
			signalDefinitions, err := w.storage.GetSignalDefinitions()
			if err != nil {
				log.Printf("Failed to load signal definitions: %v", err)
				// Fallback to empty strategies to avoid crash
				strategyConfigs = []types.Config{}
			} else {
				signalDefMap := make(map[string]storage.SignalDefinition)
				for _, def := range signalDefinitions {
					signalDefMap[def.ID] = def
				}
				
				strategyConfigs = make([]types.Config, 0, len(runConfig.SignalsWithParams))
				
				for _, signalWithParams := range runConfig.SignalsWithParams {
					if def, exists := signalDefMap[signalWithParams.SignalDefinitionID]; exists {
						log.Printf("Converting signal definition '%s' (type: %s) with parameters: %+v", def.Name, def.Type, signalWithParams.Parameters)
						
						// Use SignalAdapter for all signals in the new format
						strategyConfigs = append(strategyConfigs, types.Config{
							Type:         "SignalAdapter",
							Symbol:       runConfig.Symbol,
							Params:       types.Params{Values: []string{def.Type}}, // Pass signal type from definition
							SignalParams: signalWithParams.Parameters, // Pass signal parameters
						})
					} else {
						// Handle direct signal type IDs (like "rsi", "sma_crossover") from available signals
						log.Printf("Signal definition not found for ID: %s, treating as direct signal type", signalWithParams.SignalDefinitionID)
						
						// Use the ID directly as the signal type
						strategyConfigs = append(strategyConfigs, types.Config{
							Type:         "SignalAdapter",
							Symbol:       runConfig.Symbol,
							Params:       types.Params{Values: []string{signalWithParams.SignalDefinitionID}}, // Use ID as signal type
							SignalParams: signalWithParams.Parameters, // Pass signal parameters
						})
					}
				}
			}
		} else if len(runConfig.Signals) > 0 {
			// Handle legacy Signals format
			// Get signal definitions to extract the proper signal types
			signalDefinitions, err := w.storage.GetSignalDefinitions()
			if err != nil {
				log.Printf("Failed to load signal definitions: %v", err)
				// Fallback to empty strategies to avoid crash
				strategyConfigs = []types.Config{}
			} else {
				signalDefMap := make(map[string]storage.SignalDefinition)
				for _, def := range signalDefinitions {
					signalDefMap[def.ID] = def
				}
				
				strategyConfigs = make([]types.Config, 0, len(runConfig.Signals))
				for _, signalID := range runConfig.Signals {
					if def, exists := signalDefMap[signalID]; exists {
						// Map signal types to strategy types
						var strategyType string
						switch def.Type {
						case "rsi":
							strategyType = "RSI"
						case "sma_crossover":
							strategyType = "SignalAdapter"
						case "macd":
							strategyType = "MACD"
						default:
							log.Printf("Warning: Unknown signal type '%s', skipping signal %s", def.Type, signalID)
							continue
						}
						
						log.Printf("Converting signal '%s' of type '%s' to strategy '%s'", def.Name, def.Type, strategyType)
						
						// For SignalAdapter, pass the signal type and parameters
						if strategyType == "SignalAdapter" {
							strategyConfigs = append(strategyConfigs, types.Config{
								Type:         strategyType,
								Symbol:       runConfig.Symbol,
								Params:       types.Params{Values: []string{def.Type}}, // Pass signal type
								SignalParams: def.Parameters, // Pass signal parameters
							})
						} else {
							strategyConfigs = append(strategyConfigs, types.Config{
								Type:   strategyType,
								Symbol: runConfig.Symbol,
								Params: types.Params{}, // Use default parameters for legacy strategies
							})
						}
					} else {
						log.Printf("Warning: Signal definition not found for ID: %s", signalID)
					}
				}
			}
		}
	}
	
	return &config.Config{
		AuthConfig: w.config.AuthConfig,
		Broker:     brokerConfig,
		Datafeeds:  datafeedConfigs,
		Strategy:   strategyConfigs,
		Database:   w.config.Database,
		API:        w.config.API,
	}
}

// These converter functions should match the ones in the API package
func (w *Worker) convertBrokerConfig(cfg storage.BrokerConfig) broker.Config {
	// Ensure broker type has correct capitalization
	brokerType := cfg.Type
	if brokerType == "futures" {
		brokerType = "Futures"
	}
	
	log.Printf("Converting broker config: input type '%s' -> output type '%s'", cfg.Type, brokerType)
	
	return broker.Config{
		Type:               brokerType,
		StartingBalance:    cfg.StartingBalance,
		WeeklyWithdrawl:    cfg.WeeklyWithdrawl,
		TrailingStopAmount: cfg.TrailingStopAmount,
		FeePerSide:         cfg.FeePerSide,
		OpenSlippage:       cfg.OpenSlippage,
		TradeQuantity:      1, // Default trade quantity
		BlackoutTimes: broker.BlackoutTimesStrings{
			StartTime: "23:59", // Default to no blackout
			EndTime:   "23:59",
			TimeZone:  "UTC",
		},
		Stops:   []stop.Config{},   // No stops by default
		Profits: []profit.Config{}, // No profit targets by default
		Symbol: broker.Symbol{
			Name:       cfg.Symbol.Name,
			Margin:     cfg.Symbol.Margin,
			PointPrice: cfg.Symbol.PointPrice,
		},
	}
}

func (w *Worker) convertDatafeedConfigs(cfgs []storage.DatafeedConfig, startTime, endTime time.Time, symbol string) []datafeed.Config {
	// If no datafeeds specified, create default Database datafeed
	if len(cfgs) == 0 {
		return []datafeed.Config{
			{
				Type:      "Database",
				Symbol:    symbol,
				Interval:  "tick",
				StartTime: startTime.Unix(),
				EndTime:   endTime.Unix(),
				Storage:   w.storage,
			},
		}
	}
	
	configs := make([]datafeed.Config, len(cfgs))
	for i, cfg := range cfgs {
		configs[i] = datafeed.Config{
			Type:      cfg.Type,
			Symbol:    cfg.Symbol,
			DataPath:  cfg.DataPath,
			Interval:  cfg.Interval,
			StartTime: startTime.Unix(),
			EndTime:   endTime.Unix(),
			Storage:   w.storage, // Pass storage for Database datafeed
		}
	}
	return configs
}

func (w *Worker) convertStrategyConfigs(cfgs []storage.StrategyConfig) []types.Config {
	configs := make([]types.Config, len(cfgs))
	for i, cfg := range cfgs {
		// Convert params map to the expected format
		params := types.Params{}
		if dataPath, ok := cfg.Params["data_path"].(string); ok {
			params.DataPath = dataPath
		}
		if values, ok := cfg.Params["values"].([]string); ok {
			params.Values = values
		}
		
		configs[i] = types.Config{
			Type:   cfg.Type,
			Symbol: cfg.Symbol,
			Params: params,
		}
	}
	return configs
}