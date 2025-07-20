package jobs

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tradestax/traedor/internal/config"
	"github.com/tradestax/traedor/pkg/broker"
	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/storage"
	"github.com/tradestax/traedor/pkg/strategy/types"
)

type WorkerPool struct {
	config      *config.Config
	storage     storage.IStorage
	workers     []*Worker
	workerCount int
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
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
		config:      cfg,
		storage:     store,
		workerCount: workerCount,
		ctx:         ctx,
		cancel:      cancel,
		workers:     make([]*Worker, workerCount),
	}
	
	// Create workers
	for i := 0; i < workerCount; i++ {
		pool.workers[i] = &Worker{
			id:      fmt.Sprintf("worker-%d", i),
			config:  cfg,
			storage: store,
			ctx:     ctx,
		}
	}
	
	return pool
}

func (wp *WorkerPool) Start() {
	log.Printf("Starting worker pool with %d workers", wp.workerCount)
	
	for _, worker := range wp.workers {
		wp.wg.Add(1)
		go func(w *Worker) {
			defer wp.wg.Done()
			w.run()
		}(worker)
	}
}

func (wp *WorkerPool) Stop() {
	log.Println("Stopping worker pool...")
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
	
	if err := w.storage.UpdateRunProgress(runID, 0.0, "Initializing backtest..."); err != nil {
		log.Printf("Failed to update progress: %v", err)
	}
	
	// For now, just update progress without actual trader execution
	// TODO: Integrate with actual trader when import cycles are resolved
	// traderConfig := w.buildTraderConfig(run.Config)
	
	// Simulate backtest execution with progress updates
	steps := []struct {
		progress float64
		message  string
		duration time.Duration
	}{
		{20.0, "Loading market data...", 1 * time.Second},
		{40.0, "Processing signals...", 2 * time.Second},
		{60.0, "Executing trades...", 2 * time.Second},
		{80.0, "Calculating metrics...", 1 * time.Second},
		{100.0, "Completed successfully", 0},
	}
	
	for _, step := range steps {
		if step.duration > 0 {
			time.Sleep(step.duration)
		}
		
		if err := w.storage.UpdateRunProgress(runID, step.progress, step.message); err != nil {
			log.Printf("Failed to update progress: %v", err)
		}
		
		// Check if run was cancelled
		currentRun, err := w.storage.GetRun(runID)
		if err != nil {
			return fmt.Errorf("failed to check run status: %w", err)
		}
		if currentRun.Status == storage.RunStatusCancelled {
			return fmt.Errorf("run was cancelled")
		}
	}
	
	// Mark as completed with mock performance metrics
	mockMetrics := &storage.PerformanceMetrics{
		TotalTrades:      10,
		WinningTrades:    6,
		LosingTrades:     4,
		TotalProfit:      1000.0,
		ReturnPercentage: 10.0,
		WinRate:          60.0,
		FinalBalance:     11000.0,
	}
	
	if err := w.storage.UpdateRunStatus(runID, storage.RunStatusCompleted, mockMetrics); err != nil {
		return fmt.Errorf("failed to update completion status: %w", err)
	}
	
	log.Printf("Worker %s completed run %s successfully", w.id, runID)
	return nil
}

func (w *Worker) buildTraderConfig(runConfig storage.RunConfig) *config.Config {
	// Convert storage.RunConfig to config.Config
	brokerConfig := w.convertBrokerConfig(runConfig.Broker)
	datafeedConfigs := w.convertDatafeedConfigs(runConfig.Datafeeds)
	strategyConfigs := w.convertStrategyConfigs(runConfig.Strategies)
	
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
	return broker.Config{
		Type:               cfg.Type,
		StartingBalance:    cfg.StartingBalance,
		WeeklyWithdrawl:    cfg.WeeklyWithdrawl,
		TrailingStopAmount: cfg.TrailingStopAmount,
		FeePerSide:         cfg.FeePerSide,
		OpenSlippage:       cfg.OpenSlippage,
		Symbol: broker.Symbol{
			Name:       cfg.Symbol.Name,
			Margin:     cfg.Symbol.Margin,
			PointPrice: cfg.Symbol.PointPrice,
		},
	}
}

func (w *Worker) convertDatafeedConfigs(cfgs []storage.DatafeedConfig) []datafeed.Config {
	configs := make([]datafeed.Config, len(cfgs))
	for i, cfg := range cfgs {
		configs[i] = datafeed.Config{
			Type:     cfg.Type,
			Symbol:   cfg.Symbol,
			DataPath: cfg.DataPath,
			Interval: cfg.Interval,
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