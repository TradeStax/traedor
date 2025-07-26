package storage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"
)

// CreateMarketDataFile creates a new market data file record
func (p *PostgresStorage) CreateMarketDataFile(file *MarketDataFile) (string, error) {
	query := `
		INSERT INTO market_data_files (
			filename, file_path, file_size, file_hash, status,
			progress_percentage, lines_processed, total_lines, processing_start_time
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`
	
	var id string
	err := p.db.QueryRow(
		query, 
		file.Filename, 
		file.FilePath, 
		file.FileSize, 
		file.FileHash, 
		file.Status,
		file.ProgressPercentage,
		file.LinesProcessed,
		file.TotalLines,
		file.ProcessingStartTime,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("failed to create market data file: %w", err)
	}
	
	return id, nil
}

// UpdateMarketDataFileStatus updates the status of a market data file
func (p *PostgresStorage) UpdateMarketDataFileStatus(fileID, status, message string) error {
	query := `
		UPDATE market_data_files 
		SET status = $1, status_message = $2, imported_at = $3
		WHERE id = $4
	`
	
	var importedAt *time.Time
	if status == "completed" {
		now := time.Now()
		importedAt = &now
	}
	
	_, err := p.db.Exec(query, status, message, importedAt, fileID)
	if err != nil {
		return fmt.Errorf("failed to update file status: %w", err)
	}
	
	return nil
}

// UpdateMarketDataFileRowCount updates the row count of imported data
func (p *PostgresStorage) UpdateMarketDataFileRowCount(fileID string, rowCount int) error {
	query := `UPDATE market_data_files SET row_count = $1 WHERE id = $2`
	
	_, err := p.db.Exec(query, rowCount, fileID)
	if err != nil {
		return fmt.Errorf("failed to update row count: %w", err)
	}
	
	return nil
}

// FileAlreadyImported checks if a file with the given hash already exists and was successfully completed
func (p *PostgresStorage) FileAlreadyImported(hash string) (bool, error) {
	query := `SELECT COUNT(*) FROM market_data_files WHERE file_hash = $1 AND status = 'completed'`
	
	var count int
	err := p.db.QueryRow(query, hash).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check file existence: %w", err)
	}
	
	return count > 0, nil
}

// ListMarketDataFiles returns all market data files
func (p *PostgresStorage) ListMarketDataFiles() ([]*MarketDataFile, error) {
	query := `
		SELECT id, filename, file_path, file_size, file_hash, status, 
		       status_message, row_count, imported_at, created_at, updated_at,
		       progress_percentage, lines_processed, total_lines, processing_start_time,
		       estimated_completion_time, processing_rate, current_batch, total_batches,
		       last_processed_line_preview, error_count
		FROM market_data_files
		ORDER BY created_at DESC
	`
	
	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to list market data files: %w", err)
	}
	defer rows.Close()
	
	var files []*MarketDataFile
	for rows.Next() {
		var f MarketDataFile
		var statusMsg sql.NullString
		var rowCount sql.NullInt64
		var progressPercentage sql.NullInt64
		var linesProcessed sql.NullInt64
		var totalLines sql.NullInt64
		var processingRate sql.NullFloat64
		var currentBatch sql.NullInt64
		var totalBatches sql.NullInt64
		var linePreview sql.NullString
		var errorCount sql.NullInt64
		
		err := rows.Scan(
			&f.ID,
			&f.Filename,
			&f.FilePath,
			&f.FileSize,
			&f.FileHash,
			&f.Status,
			&statusMsg,
			&rowCount,
			&f.ImportedAt,
			&f.CreatedAt,
			&f.UpdatedAt,
			&progressPercentage,
			&linesProcessed,
			&totalLines,
			&f.ProcessingStartTime,
			&f.EstimatedCompletionTime,
			&processingRate,
			&currentBatch,
			&totalBatches,
			&linePreview,
			&errorCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		
		if statusMsg.Valid {
			f.StatusMessage = statusMsg.String
		}
		if rowCount.Valid {
			f.RowCount = int(rowCount.Int64)
		}
		if progressPercentage.Valid {
			f.ProgressPercentage = int(progressPercentage.Int64)
		}
		if linesProcessed.Valid {
			f.LinesProcessed = linesProcessed.Int64
		}
		if totalLines.Valid {
			f.TotalLines = totalLines.Int64
		}
		if processingRate.Valid {
			f.ProcessingRate = processingRate.Float64
		}
		if currentBatch.Valid {
			f.CurrentBatch = int(currentBatch.Int64)
		}
		if totalBatches.Valid {
			f.TotalBatches = int(totalBatches.Int64)
		}
		if linePreview.Valid {
			f.LastProcessedLinePreview = linePreview.String
		}
		if errorCount.Valid {
			f.ErrorCount = int(errorCount.Int64)
		}
		
		files = append(files, &f)
	}
	
	return files, nil
}

// BulkInsertOHLCData inserts multiple OHLC data records efficiently
func (p *PostgresStorage) BulkInsertOHLCData(data []OHLCData) error {
	if len(data) == 0 {
		return nil
	}
	
	// Build values for bulk insert with ON CONFLICT handling
	valueStrings := make([]string, 0, len(data))
	valueArgs := make([]interface{}, 0, len(data)*15)
	
	for i, record := range data {
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)",
			i*15+1, i*15+2, i*15+3, i*15+4, i*15+5, i*15+6, i*15+7, i*15+8,
			i*15+9, i*15+10, i*15+11, i*15+12, i*15+13, i*15+14, i*15+15))
		
		valueArgs = append(valueArgs,
			record.FileID,
			record.Symbol,
			record.Time,
			record.TickSequence,
			record.Open,
			record.High,
			record.Low,
			record.Close,
			record.Volume,
			record.TradeCount,
			record.OHLCAvg,
			record.HLCAvg,
			record.HLAvg,
			record.BidVolume,
			record.AskVolume,
		)
	}
	
	query := fmt.Sprintf(`
		INSERT INTO ohlc_data (
			file_id, symbol, time, tick_sequence, open, high, low, close,
			volume, trade_count, ohlc_avg, hlc_avg, hl_avg,
			bid_volume, ask_volume
		)
		VALUES %s
		ON CONFLICT (symbol, time, tick_sequence) DO NOTHING
	`, strings.Join(valueStrings, ","))
	
	_, err := p.db.Exec(query, valueArgs...)
	if err != nil {
		return fmt.Errorf("failed to insert OHLC data: %w", err)
	}
	
	return nil
}

// BulkInsertTechnicalIndicators inserts multiple technical indicator records
func (p *PostgresStorage) BulkInsertTechnicalIndicators(indicators []TechnicalIndicator) error {
	if len(indicators) == 0 {
		return nil
	}
	
	// Build values for bulk insert
	valueStrings := make([]string, 0, len(indicators))
	valueArgs := make([]interface{}, 0, len(indicators)*5)
	
	for i, ind := range indicators {
		// Convert indicator values to JSON
		jsonValue, err := json.Marshal(ind.IndicatorValue)
		if err != nil {
			return fmt.Errorf("failed to marshal indicator value: %w", err)
		}
		
		valueStrings = append(valueStrings, fmt.Sprintf("($%d, $%d, $%d, $%d, $%d)",
			i*5+1, i*5+2, i*5+3, i*5+4, i*5+5))
		
		valueArgs = append(valueArgs,
			ind.FileID,
			ind.Symbol,
			ind.Time,
			ind.IndicatorName,
			jsonValue,
		)
	}
	
	query := fmt.Sprintf(`
		INSERT INTO technical_indicators (file_id, symbol, time, indicator_name, indicator_values)
		VALUES %s
		ON CONFLICT (symbol, time, indicator_name) DO NOTHING
	`, strings.Join(valueStrings, ","))
	
	_, err := p.db.Exec(query, valueArgs...)
	if err != nil {
		return fmt.Errorf("failed to insert technical indicators: %w", err)
	}
	
	return nil
}

// GetOHLCData retrieves OHLC data for a symbol within a time range
func (p *PostgresStorage) GetOHLCData(symbol string, startTime, endTime time.Time) ([]OHLCData, error) {
	query := `
		SELECT id, file_id, symbol, time, open, high, low, close,
		       volume, trade_count, ohlc_avg, hlc_avg, hl_avg,
		       bid_volume, ask_volume, created_at
		FROM ohlc_data
		WHERE symbol = $1 AND time >= $2 AND time <= $3
		ORDER BY time ASC
	`
	
	rows, err := p.db.Query(query, symbol, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query OHLC data: %w", err)
	}
	defer rows.Close()
	
	var data []OHLCData
	for rows.Next() {
		var d OHLCData
		err := rows.Scan(
			&d.ID,
			&d.FileID,
			&d.Symbol,
			&d.Time,
			&d.Open,
			&d.High,
			&d.Low,
			&d.Close,
			&d.Volume,
			&d.TradeCount,
			&d.OHLCAvg,
			&d.HLCAvg,
			&d.HLAvg,
			&d.BidVolume,
			&d.AskVolume,
			&d.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan OHLC data: %w", err)
		}
		data = append(data, d)
	}
	
	return data, nil
}

// GetTechnicalIndicators retrieves technical indicators for a symbol
func (p *PostgresStorage) GetTechnicalIndicators(symbol string, indicatorName string, startTime, endTime time.Time) ([]TechnicalIndicator, error) {
	query := `
		SELECT id, file_id, symbol, time, indicator_name, indicator_values, created_at
		FROM technical_indicators
		WHERE symbol = $1 AND indicator_name = $2 AND time >= $3 AND time <= $4
		ORDER BY time ASC
	`
	
	rows, err := p.db.Query(query, symbol, indicatorName, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to query technical indicators: %w", err)
	}
	defer rows.Close()
	
	var indicators []TechnicalIndicator
	for rows.Next() {
		var ind TechnicalIndicator
		var jsonValue []byte
		
		err := rows.Scan(
			&ind.ID,
			&ind.FileID,
			&ind.Symbol,
			&ind.Time,
			&ind.IndicatorName,
			&jsonValue,
			&ind.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan indicator: %w", err)
		}
		
		// Unmarshal JSON value
		if err := json.Unmarshal(jsonValue, &ind.IndicatorValue); err != nil {
			return nil, fmt.Errorf("failed to unmarshal indicator value: %w", err)
		}
		
		indicators = append(indicators, ind)
	}
	
	return indicators, nil
}

// UpdateFileProgress updates the progress information for a file import
func (p *PostgresStorage) UpdateFileProgress(progress ProgressUpdate) error {
	query := `
		UPDATE market_data_files 
		SET progress_percentage = $1,
		    lines_processed = $2,
		    processing_rate = $3,
		    estimated_completion_time = $4,
		    current_batch = $5,
		    total_batches = $6,
		    last_processed_line_preview = $7
		WHERE id = $8
	`
	
	var estCompletion *time.Time
	if !progress.EstimatedCompletionTime.IsZero() {
		estCompletion = &progress.EstimatedCompletionTime
	}
	
	_, err := p.db.Exec(query,
		progress.ProgressPercentage,
		progress.LinesProcessed,
		progress.ProcessingRate,
		estCompletion,
		progress.CurrentBatch,
		progress.TotalBatches,
		progress.LastProcessedLinePreview,
		progress.FileID,
	)
	if err != nil {
		return fmt.Errorf("failed to update file progress: %w", err)
	}
	
	return nil
}

// DeleteMarketDataFile deletes a specific market data file record
func (p *PostgresStorage) DeleteMarketDataFile(fileID string) error {
	_, err := p.db.Exec("DELETE FROM market_data_files WHERE id = $1", fileID)
	if err != nil {
		return fmt.Errorf("failed to delete market data file: %w", err)
	}
	return nil
}

// DeleteFailedImports deletes all failed import records and returns the count
func (p *PostgresStorage) DeleteFailedImports() (int, error) {
	result, err := p.db.Exec("DELETE FROM market_data_files WHERE status = 'failed'")
	if err != nil {
		return 0, fmt.Errorf("failed to delete failed imports: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return int(rowsAffected), nil
}

// DeletePendingImports deletes all pending import records
func (p *PostgresStorage) DeletePendingImports() (int, error) {
	result, err := p.db.Exec("DELETE FROM market_data_files WHERE status = 'pending'")
	if err != nil {
		return 0, fmt.Errorf("failed to delete pending imports: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return int(rowsAffected), nil
}

// UpdateMarketDataFileTotalLines updates the total lines count for a file after streaming import
func (p *PostgresStorage) UpdateMarketDataFileTotalLines(fileID string, totalLines int64) error {
	query := `
		UPDATE market_data_files 
		SET total_lines = $1,
		    progress_percentage = CASE 
		        WHEN total_lines > 0 THEN (lines_processed * 100) / $1
		        ELSE 0 
		    END,
		    updated_at = NOW()
		WHERE id = $2
	`
	
	_, err := p.db.Exec(query, totalLines, fileID)
	if err != nil {
		return fmt.Errorf("failed to update total lines: %w", err)
	}
	
	return nil
}

// ResetStuckImports resets all imports stuck in processing or pending state
func (p *PostgresStorage) ResetStuckImports() (int, error) {
	query := `
		UPDATE market_data_files 
		SET status = 'failed',
		    status_message = 'Reset due to server restart',
		    updated_at = NOW()
		WHERE status IN ('processing', 'pending')
	`
	
	result, err := p.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to reset stuck imports: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	return int(rowsAffected), nil
}

// GetStuckImports returns files that were in processing or pending state
func (p *PostgresStorage) GetStuckImports() ([]*MarketDataFile, error) {
	query := `
		SELECT id, filename, file_path, file_size, file_hash, status, 
		       status_message, row_count, imported_at, created_at, updated_at,
		       progress_percentage, lines_processed, total_lines, processing_start_time,
		       estimated_completion_time, processing_rate, current_batch, total_batches,
		       last_processed_line_preview, error_count
		FROM market_data_files
		WHERE status IN ('processing', 'pending')
		ORDER BY created_at ASC
	`
	
	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get stuck imports: %w", err)
	}
	defer rows.Close()
	
	var files []*MarketDataFile
	for rows.Next() {
		var f MarketDataFile
		var statusMsg sql.NullString
		var rowCount sql.NullInt64
		var progressPercentage sql.NullInt64
		var linesProcessed sql.NullInt64
		var totalLines sql.NullInt64
		var processingRate sql.NullFloat64
		var currentBatch sql.NullInt64
		var totalBatches sql.NullInt64
		var linePreview sql.NullString
		var errorCount sql.NullInt64
		
		err := rows.Scan(
			&f.ID,
			&f.Filename,
			&f.FilePath,
			&f.FileSize,
			&f.FileHash,
			&f.Status,
			&statusMsg,
			&rowCount,
			&f.ImportedAt,
			&f.CreatedAt,
			&f.UpdatedAt,
			&progressPercentage,
			&linesProcessed,
			&totalLines,
			&f.ProcessingStartTime,
			&f.EstimatedCompletionTime,
			&processingRate,
			&currentBatch,
			&totalBatches,
			&linePreview,
			&errorCount,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan file: %w", err)
		}
		
		if statusMsg.Valid {
			f.StatusMessage = statusMsg.String
		}
		if rowCount.Valid {
			f.RowCount = int(rowCount.Int64)
		}
		if progressPercentage.Valid {
			f.ProgressPercentage = int(progressPercentage.Int64)
		}
		if linesProcessed.Valid {
			f.LinesProcessed = linesProcessed.Int64
		}
		if totalLines.Valid {
			f.TotalLines = totalLines.Int64
		}
		if processingRate.Valid {
			f.ProcessingRate = processingRate.Float64
		}
		if currentBatch.Valid {
			f.CurrentBatch = int(currentBatch.Int64)
		}
		if totalBatches.Valid {
			f.TotalBatches = int(totalBatches.Int64)
		}
		if linePreview.Valid {
			f.LastProcessedLinePreview = linePreview.String
		}
		if errorCount.Valid {
			f.ErrorCount = int(errorCount.Int64)
		}
		
		files = append(files, &f)
	}
	
	return files, nil
}

// GetOHLCDataStream retrieves OHLC data in chunks to prevent memory issues
func (p *PostgresStorage) GetOHLCDataStream(symbol string, startTime, endTime time.Time, chunkSize int, callback func([]OHLCData) error) error {
	if chunkSize <= 0 {
		chunkSize = 1000 // Default chunk size
	}
	
	offset := 0
	
	for {
		query := `
			SELECT id, file_id, symbol, time, open, high, low, close,
			       volume, trade_count, ohlc_avg, hlc_avg, hl_avg,
			       bid_volume, ask_volume, created_at
			FROM ohlc_data
			WHERE symbol = $1 AND time >= $2 AND time <= $3
			ORDER BY time ASC
			LIMIT $4 OFFSET $5
		`
		
		rows, err := p.db.Query(query, symbol, startTime, endTime, chunkSize, offset)
		if err != nil {
			return fmt.Errorf("failed to query OHLC data chunk: %w", err)
		}
		
		var chunk []OHLCData
		for rows.Next() {
			var d OHLCData
			err := rows.Scan(
				&d.ID,
				&d.FileID,
				&d.Symbol,
				&d.Time,
				&d.Open,
				&d.High,
				&d.Low,
				&d.Close,
				&d.Volume,
				&d.TradeCount,
				&d.OHLCAvg,
				&d.HLCAvg,
				&d.HLAvg,
				&d.BidVolume,
				&d.AskVolume,
				&d.CreatedAt,
			)
			if err != nil {
				rows.Close()
				return fmt.Errorf("failed to scan OHLC data: %w", err)
			}
			chunk = append(chunk, d)
		}
		rows.Close()
		
		// If no data in this chunk, we're done
		if len(chunk) == 0 {
			break
		}
		
		// Process the chunk via callback
		if err := callback(chunk); err != nil {
			return fmt.Errorf("callback error: %w", err)
		}
		
		// If we got less than chunkSize, we're done
		if len(chunk) < chunkSize {
			break
		}
		
		// Clear the chunk to help GC (after checking length)
		chunk = nil
		
		offset += chunkSize
	}
	
	return nil
}

// GetExistingFileRecord checks if a file with the given hash already exists and returns the record
func (p *PostgresStorage) GetExistingFileRecord(hash string) (*MarketDataFile, error) {
	query := `
		SELECT id, filename, file_path, file_size, file_hash, status, 
		       status_message, row_count, imported_at, created_at, updated_at,
		       progress_percentage, lines_processed, total_lines, processing_start_time,
		       estimated_completion_time, processing_rate, current_batch, total_batches,
		       last_processed_line_preview, error_count
		FROM market_data_files
		WHERE file_hash = $1
		ORDER BY created_at DESC
		LIMIT 1
	`
	
	var f MarketDataFile
	var statusMsg sql.NullString
	var rowCount sql.NullInt64
	var progressPercentage sql.NullInt64
	var linesProcessed sql.NullInt64
	var totalLines sql.NullInt64
	var processingRate sql.NullFloat64
	var currentBatch sql.NullInt64
	var totalBatches sql.NullInt64
	var linePreview sql.NullString
	var errorCount sql.NullInt64
	
	err := p.db.QueryRow(query, hash).Scan(
		&f.ID,
		&f.Filename,
		&f.FilePath,
		&f.FileSize,
		&f.FileHash,
		&f.Status,
		&statusMsg,
		&rowCount,
		&f.ImportedAt,
		&f.CreatedAt,
		&f.UpdatedAt,
		&progressPercentage,
		&linesProcessed,
		&totalLines,
		&f.ProcessingStartTime,
		&f.EstimatedCompletionTime,
		&processingRate,
		&currentBatch,
		&totalBatches,
		&linePreview,
		&errorCount,
	)
	
	if err == sql.ErrNoRows {
		return nil, nil // No existing record found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get existing file record: %w", err)
	}
	
	// Handle nullable fields
	if statusMsg.Valid {
		f.StatusMessage = statusMsg.String
	}
	if rowCount.Valid {
		f.RowCount = int(rowCount.Int64)
	}
	if progressPercentage.Valid {
		f.ProgressPercentage = int(progressPercentage.Int64)
	}
	if linesProcessed.Valid {
		f.LinesProcessed = linesProcessed.Int64
	}
	if totalLines.Valid {
		f.TotalLines = totalLines.Int64
	}
	if processingRate.Valid {
		f.ProcessingRate = processingRate.Float64
	}
	if currentBatch.Valid {
		f.CurrentBatch = int(currentBatch.Int64)
	}
	if totalBatches.Valid {
		f.TotalBatches = int(totalBatches.Int64)
	}
	if linePreview.Valid {
		f.LastProcessedLinePreview = linePreview.String
	}
	if errorCount.Valid {
		f.ErrorCount = int(errorCount.Int64)
	}
	
	return &f, nil
}

// CountOHLCDataByFileID returns the count of OHLC data records for a specific file ID
func (p *PostgresStorage) CountOHLCDataByFileID(fileID string) (int64, error) {
	query := `SELECT COUNT(*) FROM ohlc_data WHERE file_id = $1`
	
	var count int64
	err := p.db.QueryRow(query, fileID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count OHLC data for file ID %s: %w", fileID, err)
	}
	
	return count, nil
}

// DeleteOHLCDataByFileID deletes all OHLC data records for a specific file ID
func (p *PostgresStorage) DeleteOHLCDataByFileID(fileID string) error {
	query := `DELETE FROM ohlc_data WHERE file_id = $1`
	
	result, err := p.db.Exec(query, fileID)
	if err != nil {
		return fmt.Errorf("failed to delete OHLC data for file ID %s: %w", fileID, err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	log.Printf("Deleted %d OHLC records for file ID: %s", rowsAffected, fileID)
	return nil
}