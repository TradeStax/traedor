package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/lib/pq"
	"github.com/tradestax/traedor/pkg/broker"
	"github.com/tradestax/traedor/pkg/datafeed"
)

type PostgresStorage struct {
	db *sql.DB
}

func NewPostgresStorage(connectionString string) (*PostgresStorage, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return &PostgresStorage{db: db}, nil
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
		SELECT id, config, status, started_at, completed_at, performance_metrics, created_at, updated_at
		FROM runs
		WHERE id = $1
	`

	err := p.db.QueryRow(query, runID).Scan(
		&run.ID,
		&configJSON,
		&run.Status,
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
		SELECT id, config, status, started_at, completed_at, performance_metrics, created_at, updated_at
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
	var metricsJSON []byte
	var err error

	if metrics != nil {
		metricsJSON, err = json.Marshal(metrics)
		if err != nil {
			return fmt.Errorf("failed to marshal metrics: %w", err)
		}
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

func (p *PostgresStorage) SaveTrade(runID string, trade *broker.Trade) error {
	query := `
		INSERT INTO trades (
			run_id, symbol, operation, quantity, open_price, close_price,
			open_time, close_time, net_profit, max_profit, max_drawdown
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	closeTime := sql.NullTime{}
	if trade.CloseTime > 0 {
		closeTime.Valid = true
		closeTime.Time = time.Unix(trade.CloseTime/1000, 0)
	}

	_, err := p.db.Exec(query,
		runID,
		trade.Symbol,
		trade.Operation,
		trade.Quantity,
		trade.OpenPrice,
		trade.ClosePrice,
		time.Unix(trade.OpenTime/1000, 0),
		closeTime,
		trade.NetProfit,
		trade.MaxProfit,
		trade.MaxDrawdown,
	)
	if err != nil {
		return fmt.Errorf("failed to save trade: %w", err)
	}

	return nil
}

func (p *PostgresStorage) GetTrades(runID string) ([]*broker.Trade, error) {
	query := `
		SELECT symbol, operation, quantity, open_price, close_price,
			   open_time, close_time, net_profit, max_profit, max_drawdown
		FROM trades
		WHERE run_id = $1
		ORDER BY open_time
	`

	rows, err := p.db.Query(query, runID)
	if err != nil {
		return nil, fmt.Errorf("failed to get trades: %w", err)
	}
	defer rows.Close()

	trades := []*broker.Trade{}
	for rows.Next() {
		var trade broker.Trade
		var closeTime sql.NullTime

		err := rows.Scan(
			&trade.Symbol,
			&trade.Operation,
			&trade.Quantity,
			&trade.OpenPrice,
			&trade.ClosePrice,
			&trade.OpenTime,
			&closeTime,
			&trade.NetProfit,
			&trade.MaxProfit,
			&trade.MaxDrawdown,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan trade: %w", err)
		}

		if closeTime.Valid {
			trade.CloseTime = closeTime.Time.Unix() * 1000
		}

		trades = append(trades, &trade)
	}

	return trades, nil
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

func (p *PostgresStorage) SaveTickData(data []datafeed.Data) error {
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
			tick.Last,
			tick.Volume,
			tick.Bid,
			tick.Ask,
		)
		if err != nil {
			return fmt.Errorf("failed to insert tick data: %w", err)
		}
	}

	return tx.Commit()
}

func (p *PostgresStorage) GetTickData(symbol string, startTime, endTime time.Time) ([]datafeed.Data, error) {
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

	data := []datafeed.Data{}
	for rows.Next() {
		var tick datafeed.Data
		var tickTime time.Time

		err := rows.Scan(
			&tickTime,
			&tick.Last,
			&tick.Volume,
			&tick.Bid,
			&tick.Ask,
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

func (p *PostgresStorage) Close() error {
	return p.db.Close()
}