package storage

import (
	"time"

	"github.com/tradestax/traedor/pkg/types"
	strategyTypes "github.com/tradestax/traedor/pkg/strategy/types"
)

type IStorage interface {
	// Run management
	CreateRun(config RunConfig) (*Run, error)
	GetRun(runID string) (*Run, error)
	ListRuns(filter RunFilter) ([]*Run, error)
	GetRunsCount(filter RunFilter) (int, error)
	UpdateRunStatus(runID string, status RunStatus, metrics *PerformanceMetrics) error
	
	// Job queue management
	UpdateRunProgress(runID string, progress float64, message string) error
	UpdateRunError(runID string, errorMsg string) error
	ClaimNextQueuedRun(workerID string) (*Run, error)
	ReleaseRunClaim(runID string) error
	RetryFailedRun(runID string) error
	CancelRun(runID string) error
	ResetStuckRuns() error

	// Trade management
	SaveTrade(runID string, trade *types.Trade) error
	GetTrades(runID string) ([]*types.Trade, error)
	GetTradesPaginated(runID string, limit, offset int) ([]*types.Trade, int, error)
	StreamTrades(runID string, callback func(*types.Trade) error) error

	// Signal management
	SaveSignal(runID string, signal Signal) error
	GetSignals(runID string) ([]*Signal, error)
	GetSignalDefinitionIDByName(name string) (string, error)

	// Tick data management
	SaveTickData(data []types.Data) error
	GetTickData(symbol string, startTime, endTime time.Time) ([]types.Data, error)

	// Signal definitions
	CreateSignalDefinition(def SignalDefinition) error
	GetSignalDefinitions() ([]SignalDefinition, error)
	UpdateSignalDefinition(id string, def SignalDefinition) error
	DeleteSignalDefinition(id string) error

	// Market data import
	CreateMarketDataFile(file *MarketDataFile) (string, error)
	UpdateMarketDataFileStatus(fileID, status, message string) error
	UpdateMarketDataFileRowCount(fileID string, rowCount int) error
	UpdateFileProgress(progress ProgressUpdate) error
	UpdateMarketDataFileTotalLines(fileID string, totalLines int64) error
	FileAlreadyImported(hash string) (bool, error)
	GetExistingFileRecord(hash string) (*MarketDataFile, error)
	ListMarketDataFiles() ([]*MarketDataFile, error)
	DeleteMarketDataFile(fileID string) error
	DeleteFailedImports() (int, error)
	DeletePendingImports() (int, error)
	BulkInsertOHLCData(data []OHLCData) error
	BulkInsertTechnicalIndicators(indicators []TechnicalIndicator) error
	GetOHLCData(symbol string, startTime, endTime time.Time) ([]OHLCData, error)
	GetOHLCDataStream(symbol string, startTime, endTime time.Time, chunkSize int, callback func([]OHLCData) error) error
	StreamOHLCData(symbol string, startTime, endTime time.Time, callback func(OHLCData) error) error
	GetTechnicalIndicators(symbol string, indicatorName string, startTime, endTime time.Time) ([]TechnicalIndicator, error)
	GetAvailableSymbols() ([]string, error)
	SymbolExists(symbol string) (bool, error)
	GetAvailableTimeframes() ([]string, error)
	GetSymbolDetails() ([]Symbol, error)
	GetImportedSymbolDetails() ([]Symbol, error)
	GetTimeframeDetails() ([]Timeframe, error)
	GetSymbolDataAvailability(symbol string) (*DataAvailability, error)
	ResetStuckImports() (int, error)
	GetStuckImports() ([]*MarketDataFile, error)
	CountOHLCDataByFileID(fileID string) (int64, error)
	DeleteOHLCDataByFileID(fileID string) error

	// Signal optimization
	CreateOptimization(config OptimizationConfig) (*Optimization, error)
	GetOptimization(optimizationID string) (*Optimization, error)
	ListOptimizations(filter OptimizationFilter) ([]*Optimization, error)
	UpdateOptimizationStatus(optimizationID string, status OptimizationStatus, progress float64, message string) error
	UpdateOptimizationResults(optimizationID string, results *OptimizationResults) error
	CancelOptimization(optimizationID string) error
	PauseOptimization(optimizationID string) error
	ResumeOptimization(optimizationID string) error
	ClaimNextQueuedOptimization(workerID string) (*Optimization, error)
	ReleaseOptimizationClaim(optimizationID string) error
	ResetStuckOptimizations() error
	UpdateOptimizationSequence(optimizationID string, totalPermutations int, parameterSequence []map[string]interface{}) error
	
	// Optimization run tracking
	CreateOptimizationRun(optimizationID string, runConfig RunConfig, parameterIndex int) (*OptimizationRun, error)
	GetOptimizationRuns(optimizationID string) ([]*OptimizationRun, error)
	UpdateOptimizationRunStatus(optimizationRunID string, status RunStatus, backTestRunID string, metrics *PerformanceMetrics) error
	GetOptimizationRunResults(optimizationID string) ([]*OptimizationRunResult, error)
	CleanupDuplicateOptimizationRunResults(optimizationID string) (int, error)
	GetRunsByConfig(config RunConfig) ([]*Run, error)

	// Cleanup
	Close() error
}

type RunConfig struct {
	Symbol            string               `json:"symbol"`
	Timeframe         string               `json:"timeframe"`
	StartTime         time.Time            `json:"start_time"`
	EndTime           time.Time            `json:"end_time"`
	Datafeeds         []DatafeedConfig     `json:"datafeeds"`
	Broker            BrokerConfig         `json:"broker"`
	Strategies        []StrategyConfig     `json:"strategies"`
	Signals           []string             `json:"signals,omitempty"` // Signal IDs to include (legacy)
	SignalsWithParams []SignalWithParams   `json:"signals_with_params,omitempty"` // New format with parameters
	SignalDefinitions []SignalDefinition   `json:"signal_definitions,omitempty"` // For creating new definitions
}

type SignalWithParams struct {
	SignalDefinitionID string                 `json:"signal_definition_id"`
	Parameters         map[string]interface{} `json:"parameters"`
}

type DatafeedConfig struct {
	Type     string `json:"type"`
	Symbol   string `json:"symbol"`
	DataPath string `json:"data_path"`
	Interval string `json:"interval"`
}

type BrokerConfig struct {
	Type               string       `json:"type"`
	StartingBalance    float64      `json:"starting_balance"`
	WeeklyWithdrawl    float64      `json:"weekly_withdrawl"`
	TrailingStopAmount float64      `json:"trailing_stop_amount"`
	FeePerSide         float64      `json:"fee_per_side"`
	OpenSlippage       float64      `json:"open_slippage"`
	Symbol             SymbolConfig `json:"symbol"`
}

type SymbolConfig struct {
	Name       string  `json:"name"`
	Margin     float64 `json:"margin"`
	PointPrice float64 `json:"point_price"`
}

type StrategyConfig struct {
	Type   string                 `json:"type"`
	Symbol string                 `json:"symbol"`
	Params map[string]interface{} `json:"params"`
}

type Run struct {
	ID                string               `json:"id"`
	Config            RunConfig            `json:"config"`
	Status            RunStatus            `json:"status"`
	StatusMessage     string               `json:"status_message"`
	Progress          float64              `json:"progress"`  // 0.0 to 100.0
	StartedAt         time.Time            `json:"started_at"`
	CompletedAt       *time.Time           `json:"completed_at"`
	PerformanceMetrics *PerformanceMetrics `json:"performance_metrics"`
	CreatedAt         time.Time            `json:"created_at"`
	UpdatedAt         time.Time            `json:"updated_at"`
	// Job tracking
	WorkerID          string               `json:"worker_id"`
	RetryCount        int                  `json:"retry_count"`
	LastError         string               `json:"last_error"`
}

type RunStatus string

const (
	RunStatusPending    RunStatus = "pending"
	RunStatusQueued     RunStatus = "queued"
	RunStatusRunning    RunStatus = "running"
	RunStatusCompleted  RunStatus = "completed"
	RunStatusFailed     RunStatus = "failed"
	RunStatusCancelled  RunStatus = "cancelled"
	RunStatusRetrying   RunStatus = "retrying"
)

type PerformanceMetrics struct {
	TotalTrades       int             `json:"total_trades"`
	WinningTrades     int             `json:"winning_trades"`
	LosingTrades      int             `json:"losing_trades"`
	TotalProfit       float64         `json:"total_profit"`
	MaxDrawdown       float64         `json:"max_drawdown"`
	MaxDrawdownPercent float64        `json:"max_drawdown_percent"`
	SharpeRatio       float64         `json:"sharpe_ratio"`
	WinRate           float64         `json:"win_rate"`
	AverageWin        float64         `json:"average_win"`
	AverageLoss       float64         `json:"average_loss"`
	ProfitFactor      float64         `json:"profit_factor"`
	FinalBalance      float64         `json:"final_balance"`
	ReturnPercentage  float64         `json:"return_percentage"`
	AverageMFE        float64         `json:"average_mfe"`  // Average Maximum Favorable Excursion
	AverageMFEPercent float64         `json:"average_mfe_percent"`
	AverageMAE        float64         `json:"average_mae"`  // Average Maximum Adverse Excursion
	AverageMAEPercent float64         `json:"average_mae_percent"`
	BalanceHistory    []BalancePoint  `json:"balance_history"`  // Account balance over time
	DrawdownHistory   []DrawdownPoint `json:"drawdown_history"` // Drawdown over time
}

type BalancePoint struct {
	Time    time.Time `json:"time"`
	Balance float64   `json:"balance"`
}

type DrawdownPoint struct {
	Time            time.Time `json:"time"`
	Drawdown        float64   `json:"drawdown"`
	DrawdownPercent float64   `json:"drawdown_percent"`
}

type Signal struct {
	ID        string
	RunID     string
	Time      time.Time
	Symbol    string
	Direction strategyTypes.Indicator
	Price     float64
	SignalID  string // References SignalDefinition.ID
	CreatedAt time.Time
}

type SignalDefinition struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"` // "technical", "ml", "custom"
	Parameters  map[string]interface{} `json:"parameters"`
	Active      bool                   `json:"active"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type RunFilter struct {
	Symbol    string
	Status    RunStatus
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
	Offset    int
}

type MarketDataFile struct {
	ID                         string     `json:"id"`
	Filename                   string     `json:"filename"`
	FilePath                   string     `json:"file_path"`
	FileSize                   int64      `json:"file_size"`
	FileHash                   string     `json:"file_hash"`
	Status                     string     `json:"status"`
	StatusMessage              string     `json:"status_message"`
	RowCount                   int        `json:"row_count"`
	ImportedAt                 *time.Time `json:"imported_at"`
	CreatedAt                  time.Time  `json:"created_at"`
	UpdatedAt                  time.Time  `json:"updated_at"`
	// Progress tracking fields
	ProgressPercentage         int        `json:"progress_percentage"`
	LinesProcessed             int64      `json:"lines_processed"`
	TotalLines                 int64      `json:"total_lines"`
	ProcessingStartTime        *time.Time `json:"processing_start_time"`
	EstimatedCompletionTime    *time.Time `json:"estimated_completion_time"`
	ProcessingRate             float64    `json:"processing_rate"`
	CurrentBatch               int        `json:"current_batch"`
	TotalBatches               int        `json:"total_batches"`
	LastProcessedLinePreview   string     `json:"last_processed_line_preview"`
	ErrorCount                 int        `json:"error_count"`
}

type OHLCData struct {
	ID           int64
	FileID       string
	Symbol       string
	Time         time.Time
	TickSequence int64
	Open         float64
	High         float64
	Low          float64
	Close        float64
	Volume       int64
	TradeCount   int
	OHLCAvg      float64
	HLCAvg       float64
	HLAvg        float64
	BidVolume    int64
	AskVolume    int64
	CreatedAt    time.Time
}

type TechnicalIndicator struct {
	ID             int64
	FileID         string
	Symbol         string
	Time           time.Time
	IndicatorName  string
	IndicatorValue map[string]interface{}
	CreatedAt      time.Time
}

type ProgressUpdate struct {
	FileID                      string
	ProgressPercentage          int
	LinesProcessed              int64
	ProcessingRate              float64
	EstimatedCompletionTime     time.Time
	CurrentBatch                int
	TotalBatches                int
	LastProcessedLinePreview    string
}

type Symbol struct {
	Name         string  `json:"name"`
	Description  string  `json:"description"`
	Margin       float64 `json:"margin"`
	PointPrice   float64 `json:"point_price"`
	TickSize     float64 `json:"tick_size"`
	ContractSize int     `json:"contract_size"`
	Currency     string  `json:"currency"`
	Exchange     string  `json:"exchange"`
	Active       bool    `json:"active"`
}

type Timeframe struct {
	Value           string `json:"value"`
	Description     string `json:"description"`
	IntervalSeconds int    `json:"interval_seconds"`
	Active          bool   `json:"active"`
}

type DataAvailability struct {
	Symbol         string    `json:"symbol"`
	EarliestData   time.Time `json:"earliest_data"`
	LatestData     time.Time `json:"latest_data"`
	TotalRecords   int64     `json:"total_records"`
	AvgIntervalSec int       `json:"avg_interval_seconds"`
}

// Signal Optimization Types
type OptimizationConfig struct {
	Name              string                        `json:"name"`
	Description       string                        `json:"description"`
	BaseRunConfig     RunConfig                     `json:"base_run_config"`
	ParameterRanges   []OptimizationParameterRange  `json:"parameter_ranges"`
	RandomOrder       bool                          `json:"random_order"`
	OptimizationMetric string                       `json:"optimization_metric"` // "cumulative_return", "sharpe_ratio", etc.
}

type OptimizationParameterRange struct {
	ParameterPath  string      `json:"parameter_path"`  // e.g., "signals_with_params.0.parameters.period"
	LowerBound     interface{} `json:"lower_bound"`
	UpperBound     interface{} `json:"upper_bound"`
	Step           interface{} `json:"step"`
	ParameterType  string      `json:"parameter_type"`  // "int", "float", "string"
}

type Optimization struct {
	ID                 string                  `json:"id"`
	Config             OptimizationConfig      `json:"config"`
	Status             OptimizationStatus      `json:"status"`
	StatusMessage      string                  `json:"status_message"`
	Progress           float64                 `json:"progress"`  // 0.0 to 100.0
	TotalPermutations  int                     `json:"total_permutations"`
	CompletedRuns      int                     `json:"completed_runs"`
	FailedRuns         int                     `json:"failed_runs"`
	StartedAt          time.Time               `json:"started_at"`
	CompletedAt        *time.Time              `json:"completed_at"`
	Results            *OptimizationResults    `json:"results"`
	CreatedAt          time.Time               `json:"created_at"`
	UpdatedAt          time.Time               `json:"updated_at"`
	WorkerID           string                  `json:"worker_id"`
	ParameterSequence  []map[string]interface{} `json:"parameter_sequence"` // Pre-calculated parameter combinations
}

type OptimizationStatus string

const (
	OptimizationStatusPending    OptimizationStatus = "pending"
	OptimizationStatusQueued     OptimizationStatus = "queued"
	OptimizationStatusRunning    OptimizationStatus = "running"
	OptimizationStatusCompleted  OptimizationStatus = "completed"
	OptimizationStatusFailed     OptimizationStatus = "failed"
	OptimizationStatusCancelled  OptimizationStatus = "cancelled"
	OptimizationStatusPaused     OptimizationStatus = "paused"
)

type OptimizationResults struct {
	BestResult         *OptimizationRunResult   `json:"best_result"`
	WorstResult        *OptimizationRunResult   `json:"worst_result"`
	AverageReturn      float64                  `json:"average_return"`
	MedianReturn       float64                  `json:"median_return"`
	BestParameters     map[string]interface{}   `json:"best_parameters"`
	CompletionTime     time.Duration            `json:"completion_time"`
	TotalBacktests     int                      `json:"total_backtests"`
	SuccessfulBacktests int                     `json:"successful_backtests"`
	FailedBacktests    int                      `json:"failed_backtests"`
}

type OptimizationRun struct {
	ID               string                 `json:"id"`
	OptimizationID   string                 `json:"optimization_id"`
	ParameterIndex   int                    `json:"parameter_index"`
	Parameters       map[string]interface{} `json:"parameters"`
	RunConfig        RunConfig              `json:"run_config"`
	BacktestRunID    string                 `json:"backtest_run_id"`
	Status           RunStatus              `json:"status"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type OptimizationRunResult struct {
	OptimizationRunID  string                 `json:"optimization_run_id"`
	ParameterIndex     int                    `json:"parameter_index"`
	Parameters         map[string]interface{} `json:"parameters"`
	BacktestRunID      string                 `json:"backtest_run_id"`
	PerformanceMetrics *PerformanceMetrics    `json:"performance_metrics"`
	OptimizationScore  float64                `json:"optimization_score"` // Based on optimization_metric
	Rank               int                    `json:"rank"`
	CompletedAt        time.Time              `json:"completed_at"`
}

type OptimizationFilter struct {
	Status    OptimizationStatus
	StartDate *time.Time
	EndDate   *time.Time
	Limit     int
	Offset    int
}