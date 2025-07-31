package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// CreateOptimization creates a new optimization run
func (p *PostgresStorage) CreateOptimization(config OptimizationConfig) (*Optimization, error) {
	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config: %w", err)
	}

	var optimization Optimization
	query := `
		INSERT INTO optimizations (config, status, started_at)
		VALUES ($1, $2, $3)
		RETURNING id, config, status, status_message, progress, total_permutations, completed_runs, failed_runs, started_at, completed_at, results, created_at, updated_at, worker_id, parameter_sequence
	`

	var statusMessage, workerID sql.NullString
	var completedAt sql.NullTime
	var results, parameterSequence []byte
	
	err = p.db.QueryRow(query, configJSON, OptimizationStatusPending, time.Now()).Scan(
		&optimization.ID,
		&configJSON,
		&optimization.Status,
		&statusMessage,
		&optimization.Progress,
		&optimization.TotalPermutations,
		&optimization.CompletedRuns,
		&optimization.FailedRuns,
		&optimization.StartedAt,
		&completedAt,
		&results,
		&optimization.CreatedAt,
		&optimization.UpdatedAt,
		&workerID,
		&parameterSequence,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create optimization: %w", err)
	}

	// Handle nullable fields
	if statusMessage.Valid {
		optimization.StatusMessage = statusMessage.String
	}
	if workerID.Valid {
		optimization.WorkerID = workerID.String
	}
	if completedAt.Valid {
		optimization.CompletedAt = &completedAt.Time
	}

	if err := json.Unmarshal(configJSON, &optimization.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Handle optional JSON fields
	if results != nil {
		var optimizationResults OptimizationResults
		if err := json.Unmarshal(results, &optimizationResults); err != nil {
			return nil, fmt.Errorf("failed to unmarshal results: %w", err)
		}
		optimization.Results = &optimizationResults
	}

	if parameterSequence != nil {
		var paramSeq []map[string]interface{}
		if err := json.Unmarshal(parameterSequence, &paramSeq); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameter sequence: %w", err)
		}
		optimization.ParameterSequence = paramSeq
	}

	return &optimization, nil
}

// GetOptimization retrieves an optimization by ID
func (p *PostgresStorage) GetOptimization(optimizationID string) (*Optimization, error) {
	var optimization Optimization
	var configJSON, resultsJSON, parameterSequenceJSON []byte
	var statusMessage, workerID sql.NullString
	var completedAt sql.NullTime

	query := `
		SELECT id, config, status, status_message, progress, total_permutations, completed_runs, failed_runs, started_at, completed_at, results, created_at, updated_at, worker_id, parameter_sequence
		FROM optimizations
		WHERE id = $1
	`

	err := p.db.QueryRow(query, optimizationID).Scan(
		&optimization.ID,
		&configJSON,
		&optimization.Status,
		&statusMessage,
		&optimization.Progress,
		&optimization.TotalPermutations,
		&optimization.CompletedRuns,
		&optimization.FailedRuns,
		&optimization.StartedAt,
		&completedAt,
		&resultsJSON,
		&optimization.CreatedAt,
		&optimization.UpdatedAt,
		&workerID,
		&parameterSequenceJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("optimization not found")
		}
		return nil, fmt.Errorf("failed to get optimization: %w", err)
	}

	// Handle nullable fields
	if statusMessage.Valid {
		optimization.StatusMessage = statusMessage.String
	}
	if workerID.Valid {
		optimization.WorkerID = workerID.String
	}
	if completedAt.Valid {
		optimization.CompletedAt = &completedAt.Time
	}

	if err := json.Unmarshal(configJSON, &optimization.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if resultsJSON != nil {
		var results OptimizationResults
		if err := json.Unmarshal(resultsJSON, &results); err != nil {
			return nil, fmt.Errorf("failed to unmarshal results: %w", err)
		}
		optimization.Results = &results
	}

	if parameterSequenceJSON != nil {
		var parameterSequence []map[string]interface{}
		if err := json.Unmarshal(parameterSequenceJSON, &parameterSequence); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameter sequence: %w", err)
		}
		optimization.ParameterSequence = parameterSequence
	}

	return &optimization, nil
}

// ListOptimizations lists optimizations with filtering
func (p *PostgresStorage) ListOptimizations(filter OptimizationFilter) ([]*Optimization, error) {
	var optimizations []*Optimization
	var conditions []string
	var args []interface{}
	argIndex := 1

	query := `
		SELECT id, config, status, status_message, progress, total_permutations, completed_runs, failed_runs, started_at, completed_at, results, created_at, updated_at, worker_id, parameter_sequence
		FROM optimizations
	`

	if filter.Status != "" {
		conditions = append(conditions, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, filter.Status)
		argIndex++
	}

	if filter.StartDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at >= $%d", argIndex))
		args = append(args, *filter.StartDate)
		argIndex++
	}

	if filter.EndDate != nil {
		conditions = append(conditions, fmt.Sprintf("created_at <= $%d", argIndex))
		args = append(args, *filter.EndDate)
		argIndex++
	}

	if len(conditions) > 0 {
		query += " WHERE " + fmt.Sprintf("%s", conditions[0])
		for i := 1; i < len(conditions); i++ {
			query += " AND " + conditions[i]
		}
	}

	query += " ORDER BY created_at DESC"

	if filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	rows, err := p.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list optimizations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var optimization Optimization
		var configJSON, resultsJSON, parameterSequenceJSON []byte
		var statusMessage, workerID sql.NullString
		var completedAt sql.NullTime

		err := rows.Scan(
			&optimization.ID,
			&configJSON,
			&optimization.Status,
			&statusMessage,
			&optimization.Progress,
			&optimization.TotalPermutations,
			&optimization.CompletedRuns,
			&optimization.FailedRuns,
			&optimization.StartedAt,
			&completedAt,
			&resultsJSON,
			&optimization.CreatedAt,
			&optimization.UpdatedAt,
			&workerID,
			&parameterSequenceJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan optimization: %w", err)
		}

		// Handle nullable fields
		if statusMessage.Valid {
			optimization.StatusMessage = statusMessage.String
		}
		if workerID.Valid {
			optimization.WorkerID = workerID.String
		}
		if completedAt.Valid {
			optimization.CompletedAt = &completedAt.Time
		}

		if err := json.Unmarshal(configJSON, &optimization.Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal config: %w", err)
		}

		if resultsJSON != nil {
			var results OptimizationResults
			if err := json.Unmarshal(resultsJSON, &results); err != nil {
				return nil, fmt.Errorf("failed to unmarshal results: %w", err)
			}
			optimization.Results = &results
		}

		if parameterSequenceJSON != nil {
			var parameterSequence []map[string]interface{}
			if err := json.Unmarshal(parameterSequenceJSON, &parameterSequence); err != nil {
				return nil, fmt.Errorf("failed to unmarshal parameter sequence: %w", err)
			}
			optimization.ParameterSequence = parameterSequence
		}

		optimizations = append(optimizations, &optimization)
	}

	return optimizations, nil
}

// UpdateOptimizationStatus updates the status of an optimization
func (p *PostgresStorage) UpdateOptimizationStatus(optimizationID string, status OptimizationStatus, progress float64, message string) error {
	now := time.Now()
	
	// Set completed_at for terminal statuses (completed, failed, cancelled)
	var query string
	var args []interface{}
	if status == OptimizationStatusCompleted || status == OptimizationStatusFailed || status == OptimizationStatusCancelled {
		query = `
			UPDATE optimizations
			SET status = $1, progress = $2, status_message = $3, completed_at = $4, updated_at = $5
			WHERE id = $6
		`
		args = []interface{}{status, progress, message, now, now, optimizationID}
	} else {
		query = `
			UPDATE optimizations
			SET status = $1, progress = $2, status_message = $3, updated_at = $4
			WHERE id = $5
		`
		args = []interface{}{status, progress, message, now, optimizationID}
	}

	_, err := p.db.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to update optimization status: %w", err)
	}

	return nil
}

// UpdateOptimizationResults updates the results of an optimization
func (p *PostgresStorage) UpdateOptimizationResults(optimizationID string, results *OptimizationResults) error {
	resultsJSON, err := json.Marshal(results)
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}

	completedAt := time.Now()
	query := `
		UPDATE optimizations
		SET results = $1, completed_at = $2, updated_at = $3
		WHERE id = $4
	`

	_, err = p.db.Exec(query, resultsJSON, completedAt, completedAt, optimizationID)
	if err != nil {
		return fmt.Errorf("failed to update optimization results: %w", err)
	}

	return nil
}

// CancelOptimization cancels an optimization
func (p *PostgresStorage) CancelOptimization(optimizationID string) error {
	now := time.Now()
	query := `
		UPDATE optimizations
		SET status = $1, completed_at = $2, updated_at = $3
		WHERE id = $4 AND status IN ($5, $6, $7)
	`

	_, err := p.db.Exec(query, OptimizationStatusCancelled, now, now, optimizationID, OptimizationStatusPending, OptimizationStatusQueued, OptimizationStatusRunning)
	if err != nil {
		return fmt.Errorf("failed to cancel optimization: %w", err)
	}

	return nil
}

// PauseOptimization pauses a running optimization
func (p *PostgresStorage) PauseOptimization(optimizationID string) error {
	now := time.Now()
	query := `
		UPDATE optimizations
		SET status = $1, updated_at = $2
		WHERE id = $3 AND status IN ($4, $5)
	`
	result, err := p.db.Exec(query, OptimizationStatusPaused, now, optimizationID, OptimizationStatusQueued, OptimizationStatusRunning)
	if err != nil {
		return fmt.Errorf("failed to pause optimization: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("optimization not found or not in a pausable state")
	}
	
	return nil
}

// ResumeOptimization resumes a paused optimization
func (p *PostgresStorage) ResumeOptimization(optimizationID string) error {
	now := time.Now()
	query := `
		UPDATE optimizations
		SET status = $1, updated_at = $2
		WHERE id = $3 AND status = $4
	`
	result, err := p.db.Exec(query, OptimizationStatusQueued, now, optimizationID, OptimizationStatusPaused)
	if err != nil {
		return fmt.Errorf("failed to resume optimization: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("optimization not found or not in paused state")
	}
	
	return nil
}

// ClaimNextQueuedOptimization claims the next queued optimization for processing
func (p *PostgresStorage) ClaimNextQueuedOptimization(workerID string) (*Optimization, error) {
	tx, err := p.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	var optimization Optimization
	var configJSON, resultsJSON, parameterSequenceJSON []byte
	var statusMessage, newWorkerID sql.NullString
	var completedAt sql.NullTime

	query := `
		UPDATE optimizations
		SET status = $1, worker_id = $2, updated_at = $3
		WHERE id = (
			SELECT id FROM optimizations
			WHERE status = $4
			ORDER BY created_at ASC
			LIMIT 1
			FOR UPDATE SKIP LOCKED
		)
		RETURNING id, config, status, status_message, progress, total_permutations, completed_runs, failed_runs, started_at, completed_at, results, created_at, updated_at, worker_id, parameter_sequence
	`

	err = tx.QueryRow(query, OptimizationStatusRunning, workerID, time.Now(), OptimizationStatusQueued).Scan(
		&optimization.ID,
		&configJSON,
		&optimization.Status,
		&statusMessage,
		&optimization.Progress,
		&optimization.TotalPermutations,
		&optimization.CompletedRuns,
		&optimization.FailedRuns,
		&optimization.StartedAt,
		&completedAt,
		&resultsJSON,
		&optimization.CreatedAt,
		&optimization.UpdatedAt,
		&newWorkerID,
		&parameterSequenceJSON,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No queued optimizations
		}
		return nil, fmt.Errorf("failed to claim optimization: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Handle nullable fields
	if statusMessage.Valid {
		optimization.StatusMessage = statusMessage.String
	}
	if newWorkerID.Valid {
		optimization.WorkerID = newWorkerID.String
	}
	if completedAt.Valid {
		optimization.CompletedAt = &completedAt.Time
	}

	if err := json.Unmarshal(configJSON, &optimization.Config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	if resultsJSON != nil {
		var results OptimizationResults
		if err := json.Unmarshal(resultsJSON, &results); err != nil {
			return nil, fmt.Errorf("failed to unmarshal results: %w", err)
		}
		optimization.Results = &results
	}

	if parameterSequenceJSON != nil {
		var parameterSequence []map[string]interface{}
		if err := json.Unmarshal(parameterSequenceJSON, &parameterSequence); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameter sequence: %w", err)
		}
		optimization.ParameterSequence = parameterSequence
	}

	return &optimization, nil
}

// ReleaseOptimizationClaim releases the claim on an optimization
func (p *PostgresStorage) ReleaseOptimizationClaim(optimizationID string) error {
	query := `
		UPDATE optimizations
		SET status = $1, worker_id = NULL, updated_at = $2
		WHERE id = $3
	`

	_, err := p.db.Exec(query, OptimizationStatusQueued, time.Now(), optimizationID)
	if err != nil {
		return fmt.Errorf("failed to release optimization claim: %w", err)
	}

	return nil
}

// ResetStuckOptimizations resets any optimizations stuck in running state
func (p *PostgresStorage) ResetStuckOptimizations() error {
	result, err := p.db.Exec(`
		UPDATE optimizations 
		SET status = $1, worker_id = NULL, updated_at = $2, progress = 0.0,
		    status_message = 'Reset by worker restart'
		WHERE status = $3
	`, OptimizationStatusQueued, time.Now(), OptimizationStatusRunning)
	
	if err != nil {
		return fmt.Errorf("failed to reset stuck optimizations: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected > 0 {
		log.Printf("Reset %d stuck optimizations from 'running' to 'queued' status", rowsAffected)
	}
	
	return nil
}

// CreateOptimizationRun creates a new optimization run
func (p *PostgresStorage) CreateOptimizationRun(optimizationID string, runConfig RunConfig, parameterIndex int) (*OptimizationRun, error) {
	runConfigJSON, err := json.Marshal(runConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal run config: %w", err)
	}

	// Extract parameters from the run config for storage
	parametersJSON, err := json.Marshal(map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal parameters: %w", err)
	}

	var optimizationRun OptimizationRun
	query := `
		INSERT INTO optimization_runs (optimization_id, parameter_index, parameters, run_config, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, optimization_id, parameter_index, parameters, run_config, backtest_run_id, status, created_at, updated_at
	`

	var backTestRunID sql.NullString
	err = p.db.QueryRow(query, optimizationID, parameterIndex, parametersJSON, runConfigJSON, RunStatusPending).Scan(
		&optimizationRun.ID,
		&optimizationRun.OptimizationID,
		&optimizationRun.ParameterIndex,
		&parametersJSON,
		&runConfigJSON,
		&backTestRunID,
		&optimizationRun.Status,
		&optimizationRun.CreatedAt,
		&optimizationRun.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create optimization run: %w", err)
	}

	if backTestRunID.Valid {
		optimizationRun.BacktestRunID = backTestRunID.String
	}

	if err := json.Unmarshal(parametersJSON, &optimizationRun.Parameters); err != nil {
		return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
	}

	if err := json.Unmarshal(runConfigJSON, &optimizationRun.RunConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal run config: %w", err)
	}

	return &optimizationRun, nil
}

// GetOptimizationRuns gets all optimization runs for an optimization
func (p *PostgresStorage) GetOptimizationRuns(optimizationID string) ([]*OptimizationRun, error) {
	query := `
		SELECT id, optimization_id, parameter_index, parameters, run_config, backtest_run_id, status, created_at, updated_at
		FROM optimization_runs
		WHERE optimization_id = $1
		ORDER BY parameter_index ASC
	`

	rows, err := p.db.Query(query, optimizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get optimization runs: %w", err)
	}
	defer rows.Close()

	var runs []*OptimizationRun

	for rows.Next() {
		var run OptimizationRun
		var parametersJSON, runConfigJSON []byte
		var backTestRunID sql.NullString

		err := rows.Scan(
			&run.ID,
			&run.OptimizationID,
			&run.ParameterIndex,
			&parametersJSON,
			&runConfigJSON,
			&backTestRunID,
			&run.Status,
			&run.CreatedAt,
			&run.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan optimization run: %w", err)
		}

		if backTestRunID.Valid {
			run.BacktestRunID = backTestRunID.String
		}

		if err := json.Unmarshal(parametersJSON, &run.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}

		if err := json.Unmarshal(runConfigJSON, &run.RunConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal run config: %w", err)
		}

		runs = append(runs, &run)
	}

	return runs, nil
}

// UpdateOptimizationRunStatus updates the status of an optimization run
func (p *PostgresStorage) UpdateOptimizationRunStatus(optimizationRunID string, status RunStatus, backTestRunID string, metrics *PerformanceMetrics) error {
	query := `
		UPDATE optimization_runs
		SET status = $1, backtest_run_id = $2, updated_at = $3
		WHERE id = $4
	`

	_, err := p.db.Exec(query, status, backTestRunID, time.Now(), optimizationRunID)
	if err != nil {
		return fmt.Errorf("failed to update optimization run status: %w", err)
	}

	// If the run completed successfully, create a result record
	if status == RunStatusCompleted && metrics != nil {
		if err := p.createOptimizationRunResult(optimizationRunID, backTestRunID, metrics); err != nil {
			log.Printf("Warning: Failed to create optimization run result: %v", err)
		}
	}

	return nil
}

// createOptimizationRunResult creates a result record for a completed optimization run
func (p *PostgresStorage) createOptimizationRunResult(optimizationRunID, backTestRunID string, metrics *PerformanceMetrics) error {
	// First get the optimization run details
	var optimizationID string
	var parameterIndex int
	var parametersJSON []byte

	query := `
		SELECT optimization_id, parameter_index, parameters
		FROM optimization_runs
		WHERE id = $1
	`

	err := p.db.QueryRow(query, optimizationRunID).Scan(&optimizationID, &parameterIndex, &parametersJSON)
	if err != nil {
		return fmt.Errorf("failed to get optimization run details: %w", err)
	}

	// Check if we already have a result for this optimization and parameter index
	var existingCount int
	checkQuery := `
		SELECT COUNT(*)
		FROM optimization_run_results
		WHERE optimization_id = $1 AND parameter_index = $2
	`
	err = p.db.QueryRow(checkQuery, optimizationID, parameterIndex).Scan(&existingCount)
	if err != nil {
		return fmt.Errorf("failed to check for existing optimization run result: %w", err)
	}

	if existingCount > 0 {
		log.Printf("Optimization run result already exists for optimization %s, parameter index %d, skipping creation", optimizationID, parameterIndex)
		return nil
	}

	// Get optimization config to determine scoring metric
	optimization, err := p.GetOptimization(optimizationID)
	if err != nil {
		return fmt.Errorf("failed to get optimization config: %w", err)
	}

	// Calculate optimization score
	score := p.calculateOptimizationScore(metrics, optimization.Config.OptimizationMetric)

	metricsJSON, err := json.Marshal(metrics)
	if err != nil {
		return fmt.Errorf("failed to marshal metrics: %w", err)
	}

	insertQuery := `
		INSERT INTO optimization_run_results (optimization_id, optimization_run_id, parameter_index, parameters, backtest_run_id, performance_metrics, optimization_score)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err = p.db.Exec(insertQuery, optimizationID, optimizationRunID, parameterIndex, parametersJSON, backTestRunID, metricsJSON, score)
	if err != nil {
		return fmt.Errorf("failed to create optimization run result: %w", err)
	}

	return nil
}

// CleanupDuplicateOptimizationRunResults removes duplicate results keeping only the best one for each parameter combination
func (p *PostgresStorage) CleanupDuplicateOptimizationRunResults(optimizationID string) (int, error) {
	// Delete duplicate results, keeping only the one with the highest optimization_score for each parameter_index
	query := `
		DELETE FROM optimization_run_results
		WHERE id NOT IN (
			SELECT DISTINCT ON (parameter_index) id
			FROM optimization_run_results
			WHERE optimization_id = $1
			ORDER BY parameter_index, optimization_score DESC, created_at ASC
		) AND optimization_id = $1
	`
	
	result, err := p.db.Exec(query, optimizationID)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup duplicate optimization run results: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected > 0 {
		log.Printf("Cleaned up %d duplicate optimization run results for optimization %s", rowsAffected, optimizationID)
	}
	
	return int(rowsAffected), nil
}

// calculateOptimizationScore calculates the optimization score based on the metric
func (p *PostgresStorage) calculateOptimizationScore(metrics *PerformanceMetrics, optimizationMetric string) float64 {
	if metrics == nil {
		return -999999 // Worst possible score for failed runs
	}

	switch optimizationMetric {
	case "cumulative_return":
		return metrics.ReturnPercentage
	case "total_profit":
		return metrics.TotalProfit
	case "sharpe_ratio":
		return metrics.SharpeRatio
	case "profit_factor":
		return metrics.ProfitFactor
	case "win_rate":
		return metrics.WinRate
	case "max_drawdown":
		return -metrics.MaxDrawdownPercent // Negative because lower drawdown is better
	default:
		// Default to cumulative return
		return metrics.ReturnPercentage
	}
}

// GetOptimizationRunResults gets all results for an optimization, sorted by score
func (p *PostgresStorage) GetOptimizationRunResults(optimizationID string) ([]*OptimizationRunResult, error) {
	query := `
		SELECT 
			optimization_run_id, parameter_index, parameters, backtest_run_id, 
			performance_metrics, optimization_score, rank_position, completed_at
		FROM optimization_run_results
		WHERE optimization_id = $1
		ORDER BY optimization_score DESC, parameter_index ASC
	`

	rows, err := p.db.Query(query, optimizationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get optimization run results: %w", err)
	}
	defer rows.Close()

	var results []*OptimizationRunResult
	rank := 1

	for rows.Next() {
		var result OptimizationRunResult
		var parametersJSON, metricsJSON []byte
		var rankPosition sql.NullInt32

		err := rows.Scan(
			&result.OptimizationRunID,
			&result.ParameterIndex,
			&parametersJSON,
			&result.BacktestRunID,
			&metricsJSON,
			&result.OptimizationScore,
			&rankPosition,
			&result.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan optimization run result: %w", err)
		}

		if rankPosition.Valid {
			result.Rank = int(rankPosition.Int32)
		} else {
			result.Rank = rank
		}

		if err := json.Unmarshal(parametersJSON, &result.Parameters); err != nil {
			return nil, fmt.Errorf("failed to unmarshal parameters: %w", err)
		}

		if metricsJSON != nil {
			var metrics PerformanceMetrics
			if err := json.Unmarshal(metricsJSON, &metrics); err != nil {
				return nil, fmt.Errorf("failed to unmarshal metrics: %w", err)
			}
			result.PerformanceMetrics = &metrics
		}

		results = append(results, &result)
		rank++
	}

	return results, nil
}

// UpdateOptimizationSequence updates the parameter sequence and total permutations for an optimization
func (p *PostgresStorage) UpdateOptimizationSequence(optimizationID string, totalPermutations int, parameterSequence []map[string]interface{}) error {
	parameterSequenceJSON, err := json.Marshal(parameterSequence)
	if err != nil {
		return fmt.Errorf("failed to marshal parameter sequence: %w", err)
	}

	query := `
		UPDATE optimizations
		SET total_permutations = $1, parameter_sequence = $2, updated_at = $3
		WHERE id = $4
	`

	_, err = p.db.Exec(query, totalPermutations, parameterSequenceJSON, time.Now(), optimizationID)
	if err != nil {
		return fmt.Errorf("failed to update optimization sequence: %w", err)
	}

	return nil
}