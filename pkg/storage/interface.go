package storage

import (
	"time"

	"github.com/tradestax/traedor/pkg/broker"
	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/strategy/types"
)

type IStorage interface {
	// Run management
	CreateRun(config RunConfig) (*Run, error)
	GetRun(runID string) (*Run, error)
	ListRuns(filter RunFilter) ([]*Run, error)
	UpdateRunStatus(runID string, status RunStatus, metrics *PerformanceMetrics) error
	
	// Job queue management
	UpdateRunProgress(runID string, progress float64, message string) error
	UpdateRunError(runID string, errorMsg string) error
	ClaimNextQueuedRun(workerID string) (*Run, error)
	ReleaseRunClaim(runID string) error
	RetryFailedRun(runID string) error
	CancelRun(runID string) error

	// Trade management
	SaveTrade(runID string, trade *broker.Trade) error
	GetTrades(runID string) ([]*broker.Trade, error)

	// Signal management
	SaveSignal(runID string, signal Signal) error
	GetSignals(runID string) ([]*Signal, error)

	// Tick data management
	SaveTickData(data []datafeed.Data) error
	GetTickData(symbol string, startTime, endTime time.Time) ([]datafeed.Data, error)

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
	ListMarketDataFiles() ([]*MarketDataFile, error)
	DeleteMarketDataFile(fileID string) error
	DeleteFailedImports() (int, error)
	DeletePendingImports() (int, error)
	BulkInsertOHLCData(data []OHLCData) error
	BulkInsertTechnicalIndicators(indicators []TechnicalIndicator) error
	GetOHLCData(symbol string, startTime, endTime time.Time) ([]OHLCData, error)
	GetTechnicalIndicators(symbol string, indicatorName string, startTime, endTime time.Time) ([]TechnicalIndicator, error)
	GetAvailableSymbols() ([]string, error)
	GetAvailableTimeframes() ([]string, error)

	// Cleanup
	Close() error
}

type RunConfig struct {
	Symbol     string            `json:"symbol"`
	Timeframe  string            `json:"timeframe"`
	StartTime  time.Time         `json:"start_time"`
	EndTime    time.Time         `json:"end_time"`
	Datafeeds  []DatafeedConfig  `json:"datafeeds"`
	Broker     BrokerConfig      `json:"broker"`
	Strategies []StrategyConfig  `json:"strategies"`
	Signals    []string          `json:"signals"` // Signal IDs to include
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
	Direction types.Indicator
	Price     float64
	SignalID  string // References SignalDefinition.ID
	CreatedAt time.Time
}

type SignalDefinition struct {
	ID          string
	Name        string
	Description string
	Type        string // "technical", "ml", "custom"
	Parameters  map[string]interface{}
	Active      bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
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