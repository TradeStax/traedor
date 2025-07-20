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

	// Cleanup
	Close() error
}

type RunConfig struct {
	Symbol     string
	Timeframe  string
	StartTime  time.Time
	EndTime    time.Time
	Datafeeds  []DatafeedConfig
	Broker     BrokerConfig
	Strategies []StrategyConfig
	Signals    []string // Signal IDs to include
}

type DatafeedConfig struct {
	Type     string
	Symbol   string
	DataPath string
	Interval string
}

type BrokerConfig struct {
	Type               string
	StartingBalance    float64
	WeeklyWithdrawl    float64
	TrailingStopAmount float64
	FeePerSide         float64
	OpenSlippage       float64
	Symbol             SymbolConfig
}

type SymbolConfig struct {
	Name       string
	Margin     float64
	PointPrice float64
}

type StrategyConfig struct {
	Type   string
	Symbol string
	Params map[string]interface{}
}

type Run struct {
	ID                string
	Config            RunConfig
	Status            RunStatus
	StatusMessage     string
	Progress          float64  // 0.0 to 100.0
	StartedAt         time.Time
	CompletedAt       *time.Time
	PerformanceMetrics *PerformanceMetrics
	CreatedAt         time.Time
	UpdatedAt         time.Time
	// Job tracking
	WorkerID          string
	RetryCount        int
	LastError         string
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
	TotalTrades      int
	WinningTrades    int
	LosingTrades     int
	TotalProfit      float64
	MaxDrawdown      float64
	MaxDrawdownPercent float64
	SharpeRatio      float64
	WinRate          float64
	AverageWin       float64
	AverageLoss      float64
	ProfitFactor     float64
	FinalBalance     float64
	ReturnPercentage float64
	AverageMFE       float64  // Average Maximum Favorable Excursion
	AverageMFEPercent float64
	AverageMAE       float64  // Average Maximum Adverse Excursion
	AverageMAEPercent float64
	BalanceHistory   []BalancePoint  // Account balance over time
	DrawdownHistory  []DrawdownPoint // Drawdown over time
}

type BalancePoint struct {
	Time    time.Time
	Balance float64
}

type DrawdownPoint struct {
	Time     time.Time
	Drawdown float64
	DrawdownPercent float64
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