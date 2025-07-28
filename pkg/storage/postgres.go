package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
	"github.com/tradestax/traedor/pkg/types"
)

type PostgresStorage struct {
	db *sql.DB
	signalDefCache map[string]string // Cache for signal definition name -> ID mapping
}

func NewPostgresStorage(connectionString string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool to prevent exhaustion
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	storage := &PostgresStorage{
		db: db,
		signalDefCache: make(map[string]string),
	}
	
	// Initialize signal definitions
	if err := storage.InitializeSignalDefinitions(); err != nil {
		log.Printf("Warning: Failed to initialize signal definitions: %v", err)
		// Don't fail startup, just log the error
	}

	// Populate signal definition cache
	if err := storage.populateSignalDefCache(); err != nil {
		log.Printf("Warning: Failed to populate signal definition cache: %v", err)
	}

	return storage, nil
}

func (p *PostgresStorage) CreateRun(config RunConfig) (*Run, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var run Run
	query := `
		INSERT INTO runs (config, status, started_at)
		VALUES ($1, $2, $3)
		RETURNING id, config, status, started_at, completed_at, performance_metrics, created_at, updated_at
	`

	err = p.db.QueryRow(query, configJSON, RunStatusPending, time.Now()).Scan(
		&run.ID,
		&configJSON,
		&run.Status,
		&run.StartedAt,
		&run.CompletedAt,
		&run.PerformanceMetrics,
		&run.CreatedAt,
		&run.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create run: %w", err)
	}

	if err := json.Unmarshal(configJSON, &run.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	return &run, nil
}

func (p *PostgresStorage) GetRun(runID string) (*Run, error) {
	var run Run
	var configJSON, metricsJSON []byte

	query := `
		SELECT id, config, status, status_message, progress, started_at, completed_at, performance_metrics, created_at, updated_at
		FROM runs
		WHERE id = $1
	`

	err := p.db.QueryRow(query, runID).Scan(
		&run.ID,
		&configJSON,
		&run.Status,
		&run.StatusMessage,
		&run.Progress,
		&run.StartedAt,
		&run.CompletedAt,
		&metricsJSON,
		&run.CreatedAt,
		&run.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("run not found: %s", runID)
		}
		return nil, fmt.Errorf("failed to get run: %w", err)
	}

	if err := json.Unmarshal(configJSON, &run.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if metricsJSON != nil {
		var metrics PerformanceMetrics
		if err := json.Unmarshal(metricsJSON, &metrics); err != nil {
			return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
		}
		run.PerformanceMetrics = &metrics
	}

	return &run, nil
}

func (p *PostgresStorage) ListRuns(filter RunFilter) ([]*Run, error) {
	query := `
		SELECT id, config, status, status_message, progress, started_at, completed_at, performance_metrics, created_at, updated_at
		FROM runs
		WHERE 1=1
	`
	args := []interface{}{}
	argCount := 0

	if filter.Symbol != "" {
		argCount++
		query += fmt.Sprintf(" AND config->>'symbol' = $%d", argCount)
		args = append(args, filter.Symbol)
	}

	if filter.Status != "" {
		argCount++
		query += fmt.Sprintf(" AND status = $%d", argCount)
		args = append(args, filter.Status)
	}

	if filter.StartDate != nil {
		argCount++
		query += fmt.Sprintf(" AND created_at >= $%d", argCount)
		args = append(args, *filter.StartDate)
	}

	if filter.EndDate != nil {
		argCount++
		query += fmt.Sprintf(" AND created_at <= $%d", argCount)
		args = append(args, *filter.EndDate)
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		argCount++
		query += fmt.Sprintf(" LIMIT $%d", argCount)
		args = append(args, filter.Limit)
	}

	if filter.Offset > 0 {
		argCount++
		query += fmt.Sprintf(" OFFSET $%d", argCount)
		args = append(args, filter.Offset)
	}

	rows, err := p.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list runs: %w", err)
	}
	defer rows.Close()

	runs := []*Run{}
	for rows.Next() {
		var run Run
		var configJSON, metricsJSON []byte

		err := rows.Scan(
			&run.ID,
			&configJSON,
			&run.Status,
			&run.StatusMessage,
			&run.Progress,
			&run.StartedAt,
			&run.CompletedAt,
			&metricsJSON,
			&run.CreatedAt,
			&run.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan run: %w", err)
		}

		if err := json.Unmarshal(configJSON, &run.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		if metricsJSON != nil {
			var metrics PerformanceMetrics
			if err := json.Unmarshal(metricsJSON, &metrics); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
			}
			run.PerformanceMetrics = &metrics
		}

		runs = append(runs, &run)
	}

	return runs, nil
}

func (p *PostgresStorage) UpdateRunStatus(runID string, status RunStatus, metrics *PerformanceMetrics) error {
	var metricsJSON sql.NullString
	var err error

	if metrics != nil {
		jsonBytes, err := json.Marshal(metrics)
		if err != nil {
			return fmt.Errorf("failed to marshal metrics: %w", err)
		}
		metricsJSON.String = string(jsonBytes)
		metricsJSON.Valid = true
	}

	query := `
		UPDATE runs
		SET status = $1, performance_metrics = $2, completed_at = $3
		WHERE id = $4
	`

	completedAt := sql.NullTime{}
	if status == RunStatusCompleted || status == RunStatusFailed {
		completedAt.Valid = true
		completedAt.Time = time.Now()
	}

	_, err = p.db.Exec(query, status, metricsJSON, completedAt, runID)
	if err != nil {
		return fmt.Errorf("failed to update run status: %w", err)
	}

	return nil
}

func (p *PostgresStorage) SaveTrade(runID string, trade *types.Trade) error {
	query := `
		INSERT INTO trades (
			run_id, symbol, operation, quantity, open_price, close_price,
			open_time, close_time, net_profit, max_profit, max_drawdown,
			mfe, mae, mfe_percent, mae_percent
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
	`

	closeTime := sql.NullTime{}
	if trade.CloseTime > 0 {
		closeTime.Valid = true
		closeTime.Time = time.UnixMilli(trade.CloseTime)
	}

	openTimeToStore := time.UnixMilli(trade.OpenTime)
	
	_, err := p.db.Exec(query,
		runID,
		trade.Symbol,
		trade.Operation,
		trade.Quantity,
		trade.OpenPrice,
		trade.ClosePrice,
		openTimeToStore,
		closeTime,
		trade.NetProfit,
		trade.MaxProfit,
		trade.MaxDrawdown,
		trade.MFE,
		trade.MAE,
		trade.MFEPercent,
		trade.MAEPercent,
	)
	if err != nil {
		return fmt.Errorf("failed to save trade: %w", err)
	}

	return nil
}

func (p *PostgresStorage) GetTrades(runID string) ([]*types.Trade, error) {
	query := `
		SELECT symbol, operation, quantity, open_price, close_price,
			   open_time, close_time, net_profit, max_profit, max_drawdown,
			   mfe, mae, mfe_percent, mae_percent
		FROM trades
		WHERE run_id = $1
		ORDER BY open_time
		LIMIT 5000
	`

	rows, err := p.db.Query(query, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trades: %w", err)
	}
	defer rows.Close()

	trades := []*types.Trade{}
	for rows.Next() {
		var trade types.Trade
		var openTime time.Time
		var closeTime sql.NullTime

		err := rows.Scan(
			&trade.Symbol,
			&trade.Operation,
			&trade.Quantity,
			&trade.OpenPrice,
			&trade.ClosePrice,
			&openTime,
			&closeTime,
			&trade.NetProfit,
			&trade.MaxProfit,
			&trade.MaxDrawdown,
			&trade.MFE,
			&trade.MAE,
			&trade.MFEPercent,
			&trade.MAEPercent,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trade: %w", err)
		}

		// Convert time.Time to Unix milliseconds  
		trade.OpenTime = openTime.UnixMilli()
		
		if closeTime.Valid {
			trade.CloseTime = closeTime.Time.UnixMilli()
		}

		trades = append(trades, &trade)
	}

	return trades, nil
}

func (p *PostgresStorage) GetTradesPaginated(runID string, limit, offset int) ([]*types.Trade, int, error) {
	// First get total count
	countQuery := `SELECT COUNT(*) FROM trades WHERE run_id = $1`
	var total int
	err := p.db.QueryRow(countQuery, runID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get trades count: %w", err)
	}

	// Then get paginated trades
	query := `
		SELECT symbol, operation, quantity, open_price, close_price,
			   open_time, close_time, net_profit, max_profit, max_drawdown,
			   mfe, mae, mfe_percent, mae_percent
		FROM trades
		WHERE run_id = $1
		ORDER BY open_time
		LIMIT $2 OFFSET $3
	`

	rows, err := p.db.Query(query, runID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get trades: %w", err)
	}
	defer rows.Close()

	trades := []*types.Trade{}
	for rows.Next() {
		var trade types.Trade
		var openTime time.Time
		var closeTime sql.NullTime

		err := rows.Scan(
			&trade.Symbol,
			&trade.Operation,
			&trade.Quantity,
			&trade.OpenPrice,
			&trade.ClosePrice,
			&openTime,
			&closeTime,
			&trade.NetProfit,
			&trade.MaxProfit,
			&trade.MaxDrawdown,
			&trade.MFE,
			&trade.MAE,
			&trade.MFEPercent,
			&trade.MAEPercent,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan trade: %w", err)
		}

		// Convert time.Time to Unix milliseconds  
		trade.OpenTime = openTime.UnixMilli()
		
		if closeTime.Valid {
			trade.CloseTime = closeTime.Time.UnixMilli()
		}

		trades = append(trades, &trade)
	}

	return trades, total, nil
}

// StreamTrades streams all trades for a run ID using proper streaming without batching
func (p *PostgresStorage) StreamTrades(runID string, callback func(*types.Trade) error) error {
	query := `
		SELECT symbol, operation, quantity, open_price, close_price,
			   open_time, close_time, net_profit, max_profit, max_drawdown,
			   mfe, mae, mfe_percent, mae_percent
		FROM trades
		WHERE run_id = $1
		ORDER BY open_time
	`

	rows, err := p.db.Query(query, runID)
	if err != nil {
		return fmt.Errorf("failed to query trades: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var trade types.Trade
		var openTime time.Time
		var closeTime sql.NullTime

		err := rows.Scan(
			&trade.Symbol,
			&trade.Operation,
			&trade.Quantity,
			&trade.OpenPrice,
			&trade.ClosePrice,
			&openTime,
			&closeTime,
			&trade.NetProfit,
			&trade.MaxProfit,
			&trade.MaxDrawdown,
			&trade.MFE,
			&trade.MAE,
			&trade.MFEPercent,
			&trade.MAEPercent,
		)
		if err != nil {
			return fmt.Errorf("failed to scan trade: %w", err)
		}

		// Convert time.Time to Unix milliseconds
		trade.OpenTime = openTime.UnixMilli()
		
		if closeTime.Valid {
			trade.CloseTime = closeTime.Time.UnixMilli()
		}

		// Send trade through callback function (like sending to a channel)
		if err := callback(&trade); err != nil {
			return fmt.Errorf("callback error: %w", err)
		}
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating trades: %w", err)
	}

	return nil
}

func (p *PostgresStorage) SaveSignal(runID string, signal Signal) error {
	query := `
		INSERT INTO signals (run_id, signal_definition_id, symbol, direction, price, time)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := p.db.Exec(query,
		runID,
		signal.SignalID,
		signal.Symbol,
		signal.Direction.Direction,
		signal.Price,
		signal.Time,
	)
	if err != nil {
		return fmt.Errorf("failed to save signal: %w", err)
	}

	return nil
}

func (p *PostgresStorage) GetSignals(runID string) ([]*Signal, error) {
	query := `
		SELECT id, signal_definition_id, symbol, direction, price, time, created_at
		FROM signals
		WHERE run_id = $1
		ORDER BY time
	`

	rows, err := p.db.Query(query, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get signals: %w", err)
	}
	defer rows.Close()

	signals := []*Signal{}
	for rows.Next() {
		var signal Signal
		signal.RunID = runID

		err := rows.Scan(
			&signal.ID,
			&signal.SignalID,
			&signal.Symbol,
			&signal.Direction.Direction,
			&signal.Price,
			&signal.Time,
			&signal.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan signal: %w", err)
		}

		signals = append(signals, &signal)
	}

	return signals, nil
}

func (p *PostgresStorage) SaveTickData(data []types.Data) error {
	if len(data) == 0 {
		return nil
	}

	tx, err := p.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(`
		INSERT INTO tick_data (symbol, time, price, volume, bid, ask)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (symbol, time) DO NOTHING
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, tick := range data {
		_, err := stmt.Exec(
			tick.Symbol,
			time.Unix(tick.Date/1000, 0),
			tick.Close,
			tick.Volume,
			tick.Close, // Use Close price as bid for now
			tick.Close, // Use Close price as ask for now
		)
		if err != nil {
			return fmt.Errorf("failed to insert tick data: %w", err)
		}
	}

	return tx.Commit()
}

func (p *PostgresStorage) GetTickData(symbol string, startTime, endTime time.Time) ([]types.Data, error) {
	query := `
		SELECT time, price, volume, bid, ask
		FROM tick_data
		WHERE symbol = $1 AND time >= $2 AND time <= $3
		ORDER BY time
	`

	rows, err := p.db.Query(query, symbol, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get tick data: %w", err)
	}
	defer rows.Close()

	data := []types.Data{}
	for rows.Next() {
		var tick types.Data
		var tickTime time.Time

		err := rows.Scan(
			&tickTime,
			&tick.Close,
			&tick.Volume,
			&tick.Close, // Map to Close for compatibility
			&tick.Close, // Map to Close for compatibility
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan tick data: %w", err)
		}

		tick.Symbol = symbol
		tick.Date = tickTime.Unix() * 1000
		data = append(data, tick)
	}

	return data, nil
}

func (p *PostgresStorage) CreateSignalDefinition(def SignalDefinition) error {
	paramsJSON, err := json.Marshal(def.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		INSERT INTO signal_definitions (name, description, type, parameters, active)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = p.db.Exec(query, def.Name, def.Description, def.Type, paramsJSON, def.Active)
	if err != nil {
		return fmt.Errorf("failed to create signal definition: %w", err)
	}

	return nil
}

func (p *PostgresStorage) GetSignalDefinitions() ([]SignalDefinition, error) {
	query := `
		SELECT id, name, description, type, parameters, active, created_at, updated_at
		FROM signal_definitions
		ORDER BY name
	`

	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get signal definitions: %w", err)
	}
	defer rows.Close()

	definitions := []SignalDefinition{}
	for rows.Next() {
		var def SignalDefinition
		var paramsJSON []byte

		err := rows.Scan(
			&def.ID,
			&def.Name,
			&def.Description,
			&def.Type,
			&paramsJSON,
			&def.Active,
			&def.CreatedAt,
			&def.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan signal definition: %w", err)
		}

		if err := json.Unmarshal(paramsJSON, &def.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}

		definitions = append(definitions, def)
	}

	return definitions, nil
}

func (p *PostgresStorage) UpdateSignalDefinition(id string, def SignalDefinition) error {
	paramsJSON, err := json.Marshal(def.Parameters)
	if err != nil {
		return fmt.Errorf("failed to marshal parameters: %w", err)
	}

	query := `
		UPDATE signal_definitions
		SET name = $1, description = $2, type = $3, parameters = $4, active = $5
		WHERE id = $6
	`

	_, err = p.db.Exec(query, def.Name, def.Description, def.Type, paramsJSON, def.Active, id)
	if err != nil {
		return fmt.Errorf("failed to update signal definition: %w", err)
	}

	return nil
}

func (p *PostgresStorage) DeleteSignalDefinition(id string) error {
	_, err := p.db.Exec("DELETE FROM signal_definitions WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete signal definition: %w", err)
	}
	return nil
}

// Job queue management methods
func (p *PostgresStorage) UpdateRunProgress(runID string, progress float64, message string) error {
	_, err := p.db.Exec(`
		UPDATE runs 
		SET progress = $2, status_message = $3, updated_at = NOW() 
		WHERE id = $1
	`, runID, progress, message)
	
	if err != nil {
		return fmt.Errorf("failed to update run progress: %w", err)
	}
	return nil
}

func (p *PostgresStorage) UpdateRunError(runID string, errorMsg string) error {
	_, err := p.db.Exec(`
		UPDATE runs 
		SET last_error = $2, updated_at = NOW() 
		WHERE id = $1
	`, runID, errorMsg)
	
	if err != nil {
		return fmt.Errorf("failed to update run error: %w", err)
	}
	return nil
}

func (p *PostgresStorage) ClaimNextQueuedRun(workerID string) (*Run, error) {
	tx, err := p.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()
	
	// Find and claim the next queued run atomically
	var run Run
	var configJSON []byte
	var startedAtNullable sql.NullTime
	var completedAtNullable sql.NullTime
	
	err = tx.QueryRow(`
		UPDATE runs 
		SET status = $1, worker_id = $2, started_at = NOW(), updated_at = NOW()
		WHERE id = (
			SELECT id FROM runs 
			WHERE status = $3 
			ORDER BY created_at ASC 
			LIMIT 1 
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, config, status, status_message, progress, started_at, completed_at, 
		          created_at, updated_at, worker_id, retry_count, last_error
	`, RunStatusRunning, workerID, RunStatusQueued).Scan(
		&run.ID, &configJSON, &run.Status, &run.StatusMessage, &run.Progress,
		&startedAtNullable, &completedAtNullable, &run.CreatedAt, &run.UpdatedAt,
		&run.WorkerID, &run.RetryCount, &run.LastError,
	)
	
	if err != nil {
		if err == sql.ErrNoRows {
			// No queued runs available
			return nil, nil
		}
		return nil, fmt.Errorf("failed to claim run: %w", err)
	}
	
	// Parse config JSON
	if err := json.Unmarshal(configJSON, &run.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Handle nullable times
	if startedAtNullable.Valid {
		run.StartedAt = startedAtNullable.Time
	}
	if completedAtNullable.Valid {
		run.CompletedAt = &completedAtNullable.Time
	}
	
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}
	
	log.Printf("Worker %s claimed run %s", workerID, run.ID)
	return &run, nil
}

func (p *PostgresStorage) ReleaseRunClaim(runID string) error {
	_, err := p.db.Exec(`
		UPDATE runs 
		SET worker_id = '', updated_at = NOW() 
		WHERE id = $1 AND status != $2 AND status != $3
	`, runID, RunStatusCompleted, RunStatusFailed)
	
	if err != nil {
		return fmt.Errorf("failed to release run claim: %w", err)
	}
	return nil
}

func (p *PostgresStorage) RetryFailedRun(runID string) error {
	_, err := p.db.Exec(`
		UPDATE runs 
		SET status = $1, retry_count = retry_count + 1, worker_id = '', 
		    last_error = '', progress = 0, status_message = 'Queued for retry', 
		    updated_at = NOW()
		WHERE id = $2 AND status = $3
	`, RunStatusQueued, runID, RunStatusFailed)
	
	if err != nil {
		return fmt.Errorf("failed to retry run: %w", err)
	}
	return nil
}

func (p *PostgresStorage) CancelRun(runID string) error {
	_, err := p.db.Exec(`
		UPDATE runs 
		SET status = $1, status_message = 'Cancelled by user', updated_at = NOW(), 
		    completed_at = NOW()
		WHERE id = $2 AND status IN ($3, $4, $5)
	`, RunStatusCancelled, runID, RunStatusPending, RunStatusQueued, RunStatusRunning)
	
	if err != nil {
		return fmt.Errorf("failed to cancel run: %w", err)
	}
	return nil
}

func (p *PostgresStorage) GetAvailableSymbols() ([]string, error) {
	query := `SELECT DISTINCT symbol FROM ohlc_data ORDER BY symbol`
	
	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query available symbols: %w", err)
	}
	defer rows.Close()
	
	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, fmt.Errorf("failed to scan symbol: %w", err)
		}
		symbols = append(symbols, symbol)
	}
	
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating symbols: %w", err)
	}
	
	return symbols, nil
}

func (p *PostgresStorage) SymbolExists(symbol string) (bool, error) {
	// Check both imported_symbols and ohlc_data tables
	query := `
		SELECT EXISTS(
			SELECT 1 FROM imported_symbols WHERE symbol = $1
			UNION
			SELECT 1 FROM ohlc_data WHERE symbol = $1 LIMIT 1
		)`
	
	var exists bool
	err := p.db.QueryRow(query, symbol).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check symbol existence: %w", err)
	}
	
	return exists, nil
}

func (p *PostgresStorage) GetAvailableTimeframes() ([]string, error) {
	// Analyze time intervals from the data to determine available timeframes
	query := `
		WITH intervals AS (
			SELECT 
				LEAD(time) OVER (ORDER BY time) - time AS interval_duration
			FROM ohlc_data 
			ORDER BY time 
			LIMIT 1000
		)
		SELECT DISTINCT 
			CASE 
				WHEN interval_duration = INTERVAL '1 minute' THEN '1m'
				WHEN interval_duration = INTERVAL '5 minutes' THEN '5m'
				WHEN interval_duration = INTERVAL '15 minutes' THEN '15m'
				WHEN interval_duration = INTERVAL '30 minutes' THEN '30m'
				WHEN interval_duration = INTERVAL '1 hour' THEN '1h'
				WHEN interval_duration = INTERVAL '4 hours' THEN '4h'
				WHEN interval_duration = INTERVAL '1 day' THEN '1d'
				ELSE NULL
			END as timeframe
		FROM intervals 
		WHERE interval_duration IS NOT NULL
		ORDER BY timeframe
	`
	
	rows, err := p.db.Query(query)
	if err != nil {
		// Fallback to common timeframes if query fails
		return []string{"30m", "1h", "1d"}, nil
	}
	defer rows.Close()
	
	var timeframes []string
	for rows.Next() {
		var timeframe sql.NullString
		if err := rows.Scan(&timeframe); err != nil {
			continue
		}
		if timeframe.Valid && timeframe.String != "" {
			timeframes = append(timeframes, timeframe.String)
		}
	}
	
	// If no timeframes detected, return common ones
	if len(timeframes) == 0 {
		timeframes = []string{"30m", "1h", "1d"}
	}
	
	return timeframes, nil
}

func (p *PostgresStorage) GetSymbolDetails() ([]Symbol, error) {
	// Use the optimized method that relies on imported_symbols table
	return p.GetImportedSymbolDetails()
}

// GetImportedSymbolDetails returns symbols that have imported data using the optimized imported_symbols table
func (p *PostgresStorage) GetImportedSymbolDetails() ([]Symbol, error) {
	query := `
		SELECT s.name, s.description, s.margin, s.point_price, s.tick_size, s.contract_size, s.currency, s.exchange, s.active
		FROM symbols s
		INNER JOIN imported_symbols i ON s.name = i.symbol
		WHERE s.active = true
		ORDER BY s.name
	`
	
	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query symbols: %w", err)
	}
	defer rows.Close()
	
	var symbols []Symbol
	for rows.Next() {
		var s Symbol
		err := rows.Scan(
			&s.Name, &s.Description, &s.Margin, &s.PointPrice, 
			&s.TickSize, &s.ContractSize, &s.Currency, &s.Exchange, &s.Active,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan symbol: %w", err)
		}
		symbols = append(symbols, s)
	}
	
	return symbols, rows.Err()
}

func (p *PostgresStorage) GetTimeframeDetails() ([]Timeframe, error) {
	query := `
		SELECT t.value, t.description, t.interval_seconds, t.active
		FROM timeframes t
		WHERE t.active = true
		AND (
			-- Include tick data if we have tick records
			(t.value = 'tick' AND EXISTS (SELECT 1 FROM ohlc_data LIMIT 1))
			OR
			-- Include 30m timeframe - we know this exists from earlier analysis
			(t.interval_seconds = 1800)
		)
		ORDER BY t.interval_seconds
	`
	
	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query timeframes: %w", err)
	}
	defer rows.Close()
	
	var timeframes []Timeframe
	for rows.Next() {
		var tf Timeframe
		err := rows.Scan(&tf.Value, &tf.Description, &tf.IntervalSeconds, &tf.Active)
		if err != nil {
			return nil, fmt.Errorf("failed to scan timeframe: %w", err)
		}
		timeframes = append(timeframes, tf)
	}
	
	return timeframes, rows.Err()
}

func (p *PostgresStorage) GetSymbolDataAvailability(symbol string) (*DataAvailability, error) {
	log.Printf("GetSymbolDataAvailability called with symbol: %s", symbol)
	
	// Use the optimized imported_symbols table for fast lookups
	query := `
		SELECT 
			symbol,
			earliest_data,
			latest_data,
			total_records,
			avg_interval_seconds
		FROM imported_symbols
		WHERE symbol = $1
	`
	
	var da DataAvailability
	err := p.db.QueryRow(query, symbol).Scan(
		&da.Symbol,
		&da.EarliestData,
		&da.LatestData,
		&da.TotalRecords,
		&da.AvgIntervalSec,
	)
	if err == sql.ErrNoRows {
		// Debug: Check what symbols we do have
		var symbols []string
		debugQuery := `SELECT symbol FROM imported_symbols LIMIT 10`
		rows, debugErr := p.db.Query(debugQuery)
		if debugErr == nil {
			defer rows.Close()
			for rows.Next() {
				var s string
				if scanErr := rows.Scan(&s); scanErr == nil {
					symbols = append(symbols, s)
				}
			}
			log.Printf("No data found for symbol %s. Available symbols: %v", symbol, symbols)
		}
		return nil, nil // No data available
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query data availability: %w", err)
	}
	
	return &da, nil
}

func (p *PostgresStorage) Close() error {
	return p.db.Close()
}