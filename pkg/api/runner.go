package api

import (
	"context"
	"fmt"
	"time"

	"github.com/tradestax/traedor/internal/config"
	"github.com/tradestax/traedor/pkg/broker"
	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/signals"
	"github.com/tradestax/traedor/pkg/storage"
	"github.com/tradestax/traedor/pkg/strategy"
	"github.com/tradestax/traedor/pkg/strategy/types"
	"github.com/tradestax/traedor/pkg/trader"
)

type RunManager struct {
	config         *config.Config
	storage        storage.IStorage
	runnerCh       chan RunRequest
	activeRuns     map[string]*RunContext
	signalWorkflow *signals.SignalWorkflow
}

type RunContext struct {
	RunID    string
	Config   storage.RunConfig
	Trader   *trader.Trader
	CancelFn context.CancelFunc
}

func NewRunManager(cfg *config.Config, store storage.IStorage, runnerCh chan RunRequest) *RunManager {
	return &RunManager{
		config:         cfg,
		storage:        store,
		runnerCh:       runnerCh,
		activeRuns:     make(map[string]*RunContext),
		signalWorkflow: signals.NewSignalWorkflow(store),
	}
}

func (rm *RunManager) Start(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			// Shutdown all active runs
			for _, runCtx := range rm.activeRuns {
				runCtx.CancelFn()
			}
			return
		case req := <-rm.runnerCh:
			go rm.handleRunRequest(req)
		}
	}
}

func (rm *RunManager) handleRunRequest(req RunRequest) {
	// Create run in database
	run, err := rm.storage.CreateRun(req.Config)
	if err != nil {
		req.Response <- RunResponse{Error: err}
		return
	}

	// Send immediate response with run ID
	req.Response <- RunResponse{RunID: run.ID}

	// Update status to running
	rm.storage.UpdateRunStatus(run.ID, storage.RunStatusRunning, nil)

	// Execute the backtest
	ctx, cancel := context.WithCancel(context.Background())
	runCtx := &RunContext{
		RunID:    run.ID,
		Config:   req.Config,
		CancelFn: cancel,
	}

	rm.activeRuns[run.ID] = runCtx

	// Run the backtest
	err = rm.executeBacktest(ctx, runCtx)
	
	// Update final status if there was an error (success case is handled in trader.Summary())
	if err != nil {
		rm.storage.UpdateRunStatus(run.ID, storage.RunStatusFailed, nil)
	}

	// Remove from active runs
	delete(rm.activeRuns, run.ID)
}

func (rm *RunManager) executeBacktest(ctx context.Context, runCtx *RunContext) error {
	// Initialize signal workflow for this run
	if len(runCtx.Config.Signals) > 0 {
		if err := rm.signalWorkflow.InitializeRun(runCtx.RunID, runCtx.Config.Signals); err != nil {
			return fmt.Errorf("failed to initialize signal workflow: %w", err)
		}
		defer rm.signalWorkflow.CleanupRun(runCtx.RunID)
	}

	// Build trader configuration from run config
	traderConfig := rm.buildTraderConfig(runCtx.Config)
	
	// Create datafeeds
	datafeeds, err := rm.createDatafeeds(runCtx.Config.Datafeeds)
	if err != nil {
		return fmt.Errorf("failed to create datafeeds: %w", err)
	}

	// Create broker
	broker, err := rm.createBroker(runCtx.Config.Broker)
	if err != nil {
		return fmt.Errorf("failed to create broker: %w", err)
	}

	// Create strategies
	strategies, err := rm.createStrategies(runCtx.Config.Strategies)
	if err != nil {
		return fmt.Errorf("failed to create strategies: %w", err)
	}

	// Create trader with storage integration
	traderInstance := trader.NewTraderWithStorage(traderConfig, rm.storage, runCtx.RunID)
	runCtx.Trader = traderInstance

	// Run the backtest
	if err := traderInstance.Run(); err != nil {
		return fmt.Errorf("backtest failed: %w", err)
	}

	// Calculate performance metrics and save to storage (handled in trader.Summary())
	traderInstance.Summary()
	
	return nil
}

func (rm *RunManager) buildTraderConfig(runConfig storage.RunConfig) *config.Config {
	// Build a config structure from the run configuration
	return &config.Config{
		AuthConfig: rm.config.AuthConfig,
		Broker:     rm.convertBrokerConfig(runConfig.Broker),
		Datafeeds:  rm.convertDatafeedConfigs(runConfig.Datafeeds),
		Strategy:   rm.convertStrategyConfigs(runConfig.Strategies),
		Database:   rm.config.Database,
		API:        rm.config.API,
	}
}

func (rm *RunManager) convertBrokerConfig(cfg storage.BrokerConfig) broker.Config {
	return broker.Config{
		Type:               cfg.Type,
		StartingBalance:    cfg.StartingBalance,
		WeeklyWithdrawl:    cfg.WeeklyWithdrawl,
		TrailingStopAmount: cfg.TrailingStopAmount,
		FeePerSide:         cfg.FeePerSide,
		OpenSlippage:       cfg.OpenSlippage,
		Symbol: broker.SymbolConfig{
			Name:       cfg.Symbol.Name,
			Margin:     cfg.Symbol.Margin,
			PointPrice: cfg.Symbol.PointPrice,
		},
	}
}

func (rm *RunManager) convertDatafeedConfigs(cfgs []storage.DatafeedConfig) []datafeed.Config {
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

func (rm *RunManager) convertStrategyConfigs(cfgs []storage.StrategyConfig) []types.Config {
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

func (rm *RunManager) createDatafeeds(configs []storage.DatafeedConfig) ([]datafeed.IDatafeed, error) {
	feeds := make([]datafeed.IDatafeed, 0, len(configs))
	
	for _, cfg := range configs {
		feedConfig := datafeed.Config{
			Type:     cfg.Type,
			Symbol:   cfg.Symbol,
			DataPath: cfg.DataPath,
			Interval: cfg.Interval,
		}
		
		feed, err := datafeed.Create(&feedConfig, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create datafeed %s: %w", cfg.Type, err)
		}
		
		feeds = append(feeds, feed)
	}
	
	return feeds, nil
}

func (rm *RunManager) createBroker(cfg storage.BrokerConfig) (broker.IBroker, error) {
	brokerConfig := rm.convertBrokerConfig(cfg)
	return broker.Create(&brokerConfig), nil
}

func (rm *RunManager) createStrategies(configs []storage.StrategyConfig) ([]types.IStrategy, error) {
	strategies := make([]types.IStrategy, 0, len(configs))
	
	for _, cfg := range configs {
		stratConfig := types.Config{
			Type:   cfg.Type,
			Symbol: cfg.Symbol,
		}
		
		indicatorFeed := make(chan types.Indicator, 100)
		strat := strategy.Create(&stratConfig, indicatorFeed)
		if strat == nil {
			return nil, fmt.Errorf("failed to create strategy %s", cfg.Type)
		}
		
		strategies = append(strategies, strat)
	}
	
	return strategies, nil
}


func (rm *RunManager) calculateMetrics(b broker.IBroker) *storage.PerformanceMetrics {
	// Since Summary() doesn't return data, we'll return basic metrics
	// The actual metrics calculation is now handled in the trader's Summary method
	account, _ := b.GetAccountStats()
	finalBalance := account.Balance()
	
	return &storage.PerformanceMetrics{
		TotalTrades:      0, // This will be calculated in trader
		WinningTrades:    0,
		LosingTrades:     0,
		TotalProfit:      0.0,
		MaxDrawdown:      0.0,
		SharpeRatio:      0.0,
		WinRate:          0.0,
		AverageWin:       0.0,
		AverageLoss:      0.0,
		ProfitFactor:     0.0,
		FinalBalance:     finalBalance,
		ReturnPercentage: 0.0,
	}
}