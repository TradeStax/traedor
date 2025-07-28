package importer

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/tradestax/traedor/pkg/storage"
)

const (
	batchSize = 500
	workers   = 1
)

type Importer struct {
	storage storage.IStorage
}

func NewImporter(store storage.IStorage) *Importer {
	return &Importer{
		storage: store,
	}
}

// ImportFile imports a market data file into the database
func (i *Importer) ImportFile(ctx context.Context, filePath string) error {
	// Get file info
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Calculate file hash
	hash, err := calculateFileHash(filePath)
	if err != nil {
		return fmt.Errorf("failed to calculate file hash: %w", err)
	}

	// Check if file already has a record (any status)
	existingRecord, err := i.storage.GetExistingFileRecord(hash)
	if err != nil {
		return fmt.Errorf("failed to check existing file record: %w", err)
	}
	
	var fileID string
	
	if existingRecord != nil {
		// File record exists - check status
		switch existingRecord.Status {
		case "completed":
			return fmt.Errorf("file already imported successfully")
		case "processing", "pending":
			// Resume existing import
			log.Printf("Resuming existing import for file: %s (ID: %s, Status: %s)", 
				fileInfo.Name(), existingRecord.ID, existingRecord.Status)
			fileID = existingRecord.ID
			
			// Check if data already exists for this file ID (resume scenario)
			existingDataCount, err := i.storage.CountOHLCDataByFileID(fileID)
			if err != nil {
				log.Printf("Warning: failed to count existing OHLC data: %v", err)
			} else if existingDataCount > 0 {
				// Data exists from previous import attempt - delete it to avoid duplicates
				log.Printf("Found %d existing OHLC records for file ID %s, deleting to prevent duplicates", existingDataCount, fileID)
				if err := i.storage.DeleteOHLCDataByFileID(fileID); err != nil {
					log.Printf("Warning: failed to delete existing OHLC data: %v", err)
				}
			}
			
			// Update status to processing and reset start time
			err = i.storage.UpdateMarketDataFileStatus(fileID, "processing", "Import resumed")
			if err != nil {
				log.Printf("Warning: failed to update file status to processing: %v", err)
			}
		case "failed":
			// Update existing failed record to retry
			log.Printf("Retrying failed import for file: %s (ID: %s)", fileInfo.Name(), existingRecord.ID)
			fileID = existingRecord.ID
			
			// Check if partial data exists from previous failed attempt
			existingDataCount, err := i.storage.CountOHLCDataByFileID(fileID)
			if err != nil {
				log.Printf("Warning: failed to count existing OHLC data: %v", err)
			} else if existingDataCount > 0 {
				// Partial data exists from previous failed attempt - delete it to start fresh
				log.Printf("Found %d existing OHLC records from failed import for file ID %s, deleting to start fresh", existingDataCount, fileID)
				if err := i.storage.DeleteOHLCDataByFileID(fileID); err != nil {
					log.Printf("Warning: failed to delete existing OHLC data: %v", err)
				}
			}
			
			// Reset the record for retry
			err = i.storage.UpdateMarketDataFileStatus(fileID, "processing", "Import retried")
			if err != nil {
				log.Printf("Warning: failed to update file status to processing: %v", err)
			}
		default:
			return fmt.Errorf("file has unknown status: %s", existingRecord.Status)
		}
	} else {
		// No existing record - create new one
		log.Printf("Creating new import record for file: %s", fileInfo.Name())
		now := time.Now()
		fileRecord := &storage.MarketDataFile{
			Filename:             fileInfo.Name(),
			FilePath:             filePath,
			FileSize:             fileInfo.Size(),
			FileHash:             hash,
			Status:               "processing",
			TotalLines:           0, // Will be updated as we stream
			ProcessingStartTime:  &now,
			ProgressPercentage:   0,
			LinesProcessed:       0,
		}

		fileID, err = i.storage.CreateMarketDataFile(fileRecord)
		if err != nil {
			return fmt.Errorf("failed to create file record: %w", err)
		}
	}

	// First, count the total lines for accurate progress tracking
	log.Printf("Counting total lines in file: %s", filePath)
	totalLines, err := countFileLines(filePath)
	if err != nil {
		log.Printf("Warning: failed to count lines in %s: %v", filePath, err)
		totalLines = 0 // Continue with streaming import without total
	} else {
		log.Printf("File %s contains %d lines", filePath, totalLines)
		// Update the file record with total lines
		err = i.storage.UpdateMarketDataFileTotalLines(fileID, totalLines)
		if err != nil {
			log.Printf("Warning: failed to update total lines: %v", err)
		}
	}

	// Start streaming import with progress tracking
	log.Printf("Starting streaming CSV import for file: %s (ID: %s)", filePath, fileID)
	err = i.importCSVFileStreaming(ctx, filePath, fileID)
	if err != nil {
		// Update status to failed
		log.Printf("Import failed for file %s: %v", filePath, err)
		i.storage.UpdateMarketDataFileStatus(fileID, "failed", err.Error())
		return fmt.Errorf("import failed: %w", err)
	}

	// Update status to completed
	err = i.storage.UpdateMarketDataFileStatus(fileID, "completed", "Import successful")
	if err != nil {
		return fmt.Errorf("failed to update file status: %w", err)
	}

	return nil
}

func (i *Importer) importCSVFileStreaming(ctx context.Context, filePath string, fileID string) error {
	log.Printf("Opening file for import: %s", filePath)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Detect symbol from filename
	symbol := extractSymbolFromFilename(filePath)
	log.Printf("Detected symbol: %s for file: %s", symbol, filePath)

	// Create CSV reader
	reader := csv.NewReader(bufio.NewReader(file))
	reader.TrimLeadingSpace = true

	// Read header
	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read header: %w", err)
	}
	log.Printf("CSV header: %v", header)

	// Create column index map
	colIndex := make(map[string]int)
	for i, col := range header {
		colIndex[strings.TrimSpace(col)] = i
	}

	// Streaming import - no worker pool needed
	var rowCount int64 = 0
	startTime := time.Now()
	timestampSequences := make(map[time.Time]int64)
	batch := make([]storage.OHLCData, 0, batchSize)
	
	// Cache file info to avoid repeated DB queries
	var totalLines int64 = 0
	files, _ := i.storage.ListMarketDataFiles()
	for _, f := range files {
		if f.ID == fileID {
			totalLines = f.TotalLines
			break
		}
	}
	
	log.Printf("Starting streaming import for %s", symbol)


	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read record: %w", err)
		}

		rowCount++

		// Parse timestamp
		dateStr := strings.TrimSpace(getColumn(record, colIndex, "Date"))
		timeStr := strings.TrimSpace(getColumn(record, colIndex, "Time"))
		timestamp, err := parseTimestamp(dateStr, timeStr)
		if err != nil {
			log.Printf("Warning: skipping invalid timestamp at line %d: %v", rowCount, err)
			continue
		}

		// Get tick sequence for this timestamp
		tickSeq := timestampSequences[timestamp]
		timestampSequences[timestamp] = tickSeq + 1

		// Create OHLC record
		ohlc := storage.OHLCData{
			FileID:       fileID,
			Symbol:       symbol,
			Time:         timestamp,
			TickSequence: tickSeq,
			Open:         parseFloat(getColumn(record, colIndex, "Open")),
			High:         parseFloat(getColumn(record, colIndex, "High")),
			Low:          parseFloat(getColumn(record, colIndex, "Low")),
			Close:        parseFloat(getColumn(record, colIndex, "Last")),
			Volume:       parseInt(getColumn(record, colIndex, "Volume")),
			TradeCount:   int(parseInt(getColumn(record, colIndex, "NumberOfTrades"))),
			OHLCAvg:      0,
		}

		batch = append(batch, ohlc)

		// Insert batch when full or every 1000 records for progress updates
		if len(batch) >= batchSize || rowCount%1000 == 0 {
			if len(batch) > 0 {
				log.Printf("Inserting batch of %d records at line %d", len(batch), rowCount)
				if err := i.storage.BulkInsertOHLCData(batch); err != nil {
					log.Printf("Batch insert failed at line %d: %v", rowCount, err)
					return fmt.Errorf("failed to insert batch at line %d: %w", rowCount, err)
				}
				batch = batch[:0] // Reset batch
			}

			// Update progress every 1000 records with error handling
			if rowCount%1000 == 0 {
				log.Printf("Progress update: %d lines processed", rowCount)
				if err := i.updateStreamingProgressCached(fileID, rowCount, totalLines, startTime, record); err != nil {
					log.Printf("Warning: Failed to update progress: %v", err)
				}
				
				// Periodically clean up old timestamp sequences to prevent memory growth
				if rowCount%10000 == 0 && len(timestampSequences) > 10000 {
					// Keep only recent timestamps (last 5000 entries)
					newMap := make(map[time.Time]int64, 5000)
					for k, v := range timestampSequences {
						if rowCount-int64(v) < 5000 {
							newMap[k] = v
						}
					}
					timestampSequences = newMap
				}
			}
		}
	}

	// Insert final batch
	if len(batch) > 0 {
		log.Printf("Inserting final batch of %d records", len(batch))
		if err := i.storage.BulkInsertOHLCData(batch); err != nil {
			return fmt.Errorf("failed to insert final batch: %w", err)
		}
	}

	// Final progress update with total count
	i.updateStreamingProgress(fileID, rowCount, startTime, nil)
	
	// Update total lines in file record
	err = i.storage.UpdateMarketDataFileTotalLines(fileID, rowCount)
	if err != nil {
		log.Printf("Warning: failed to update total lines: %v", err)
	}

	// Clear memory explicitly
	timestampSequences = nil
	batch = nil

	log.Printf("Streaming import completed successfully - processed %d rows", rowCount)

	return nil
}

func (i *Importer) processBatch(batch [][]string, colIndex map[string]int, fileID, symbol string) error {
	log.Printf("Processing batch of %d records for file %s", len(batch), fileID)
	ohlcData := make([]storage.OHLCData, 0, len(batch))
	indicators := make([]storage.TechnicalIndicator, 0, len(batch)*10) // Estimate ~10 indicators per row
	
	// Track tick sequences for duplicate timestamps
	timestampSequences := make(map[time.Time]int64)

	for _, record := range batch {
		// Parse timestamp
		dateStr := getColumn(record, colIndex, "Date")
		timeStr := getColumn(record, colIndex, "Time")
		timestamp, err := parseTimestamp(dateStr, timeStr)
		if err != nil {
			continue // Skip invalid timestamps
		}

		// Get tick sequence for this timestamp
		tickSeq := timestampSequences[timestamp]
		timestampSequences[timestamp] = tickSeq + 1

		// Parse OHLC data
		ohlc := storage.OHLCData{
			FileID:       fileID,
			Symbol:       symbol,
			Time:         timestamp,
			TickSequence: tickSeq,
			Open:         parseFloat(getColumn(record, colIndex, "Open")),
			High:         parseFloat(getColumn(record, colIndex, "High")),
			Low:          parseFloat(getColumn(record, colIndex, "Low")),
			Close:        parseFloat(getColumn(record, colIndex, "Last")),
			Volume:       parseInt(getColumn(record, colIndex, "Volume")),
			TradeCount:   int(parseInt(getColumn(record, colIndex, "NumberOfTrades"))),
			OHLCAvg:      0, // Not available in tick data
			HLCAvg:       0, // Not available in tick data
			HLAvg:        0, // Not available in tick data
			BidVolume:    parseInt(getColumn(record, colIndex, "BidVolume")),
			AskVolume:    parseInt(getColumn(record, colIndex, "AskVolume")),
		}
		ohlcData = append(ohlcData, ohlc)

		// Parse technical indicators
		// Moving averages
		if val := getColumn(record, colIndex, "8"); val != "" && val != "0.00" {
			indicators = append(indicators, storage.TechnicalIndicator{
				FileID:         fileID,
				Symbol:         symbol,
				Time:           timestamp,
				IndicatorName:  "SMA_8",
				IndicatorValue: map[string]interface{}{"value": parseFloat(val)},
			})
		}

		if val := getColumn(record, colIndex, "21"); val != "" && val != "0.00" {
			indicators = append(indicators, storage.TechnicalIndicator{
				FileID:         fileID,
				Symbol:         symbol,
				Time:           timestamp,
				IndicatorName:  "SMA_21",
				IndicatorValue: map[string]interface{}{"value": parseFloat(val)},
			})
		}

		// RSI
		if val := getColumn(record, colIndex, "FullK"); val != "" && val != "0.00" {
			indicators = append(indicators, storage.TechnicalIndicator{
				FileID:        fileID,
				Symbol:        symbol,
				Time:          timestamp,
				IndicatorName: "RSI",
				IndicatorValue: map[string]interface{}{
					"K": parseFloat(val),
					"D": parseFloat(getColumn(record, colIndex, "FullD")),
				},
			})
		}

		// Add other indicators as needed...
	}

	// Bulk insert
	if len(ohlcData) > 0 {
		log.Printf("Bulk inserting %d OHLC records for file %s", len(ohlcData), fileID)
		if err := i.storage.BulkInsertOHLCData(ohlcData); err != nil {
			log.Printf("Failed to bulk insert OHLC data for file %s: %v", fileID, err)
			return fmt.Errorf("failed to insert OHLC data: %w", err)
		}
		log.Printf("Successfully inserted %d OHLC records for file %s", len(ohlcData), fileID)
	}

	if len(indicators) > 0 {
		log.Printf("Bulk inserting %d indicator records for file %s", len(indicators), fileID)
		if err := i.storage.BulkInsertTechnicalIndicators(indicators); err != nil {
			log.Printf("Failed to bulk insert indicators for file %s: %v", fileID, err)
			return fmt.Errorf("failed to insert indicators: %w", err)
		}
		log.Printf("Successfully inserted %d indicator records for file %s", len(indicators), fileID)
	}

	return nil
}

// Helper functions
func calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func extractSymbolFromFilename(filePath string) string {
	// Remove path and extension
	filename := filepath.Base(filePath)
	base := strings.TrimSuffix(filename, filepath.Ext(filename))
	
	// Split by underscore or hyphen
	parts := strings.FieldsFunc(base, func(r rune) bool {
		return r == '_' || r == '-'
	})
	
	if len(parts) > 0 {
		symbol := parts[0]
		// Handle various MES contract formats
		if strings.HasPrefix(strings.ToUpper(symbol), "MES") {
			return "/MES"
		}
		// Handle ES (E-mini S&P 500)
		if strings.HasPrefix(strings.ToUpper(symbol), "ES") {
			return "/ES"
		}
	}
	return "UNKNOWN"
}

func getColumn(record []string, colIndex map[string]int, colName string) string {
	if idx, ok := colIndex[colName]; ok && idx < len(record) {
		return strings.TrimSpace(record[idx])
	}
	return ""
}

func parseTimestamp(dateStr, timeStr string) (time.Time, error) {
	// Handle both date formats:
	// Format 1: 2022-12-4 (dash separators)
	// Format 2: 2022/9/8 (slash separators)
	combined := dateStr + " " + timeStr
	
	// Try dash format first (original format)
	if t, err := time.Parse("2006-1-2 15:04:05.000000", combined); err == nil {
		return t, nil
	}
	
	// Try slash format (tick data format)
	if t, err := time.Parse("2006/1/2 15:04:05.000000", combined); err == nil {
		return t, nil
	}
	
	// Try slash format with shorter milliseconds
	if t, err := time.Parse("2006/1/2 15:04:05.000", combined); err == nil {
		return t, nil
	}
	
	// Try dash format with shorter milliseconds
	if t, err := time.Parse("2006-1-2 15:04:05.000", combined); err == nil {
		return t, nil
	}
	
	// Try slash format without milliseconds
	if t, err := time.Parse("2006/1/2 15:04:05", combined); err == nil {
		return t, nil
	}
	
	// Try dash format without milliseconds
	if t, err := time.Parse("2006-1-2 15:04:05", combined); err == nil {
		return t, nil
	}
	
	// If all formats fail, return error
	return time.Time{}, fmt.Errorf("unable to parse timestamp: %s", combined)
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

func parseInt(s string) int64 {
	if s == "" {
		return 0
	}
	val, _ := strconv.ParseInt(s, 10, 64)
	return val
}

// countFileLines counts the total number of lines in a file
func countFileLines(filePath string) (int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	lineCount := int64(0)
	for scanner.Scan() {
		lineCount++
	}

	if err := scanner.Err(); err != nil {
		return 0, err
	}

	return lineCount, nil
}

// updateProgress updates the progress information for a file import
func (i *Importer) updateProgress(fileID string, linesProcessed, totalLines int64, startTime time.Time, currentBatch, totalBatches int, lastRecord []string) {
	elapsed := time.Since(startTime)
	progressPercentage := int((linesProcessed * 100) / totalLines)
	
	// Calculate processing rate (lines per second)
	processingRate := float64(linesProcessed) / elapsed.Seconds()
	
	// Estimate completion time
	remainingLines := totalLines - linesProcessed
	var estimatedCompletion time.Time
	if processingRate > 0 {
		remainingSeconds := float64(remainingLines) / processingRate
		estimatedCompletion = time.Now().Add(time.Duration(remainingSeconds) * time.Second)
	}

	// Get a preview of the current line being processed
	var linePreview string
	if lastRecord != nil && len(lastRecord) > 0 {
		// Show first few columns as preview
		preview := make([]string, 0, 3)
		for i, col := range lastRecord {
			if i >= 3 { // Only show first 3 columns
				break
			}
			preview = append(preview, col)
		}
		linePreview = strings.Join(preview, ", ")
		if len(linePreview) > 100 {
			linePreview = linePreview[:100] + "..."
		}
	}

	// Update the database with progress information
	progressData := storage.ProgressUpdate{
		FileID:                     fileID,
		ProgressPercentage:         progressPercentage,
		LinesProcessed:             linesProcessed,
		ProcessingRate:             processingRate,
		EstimatedCompletionTime:    estimatedCompletion,
		CurrentBatch:               currentBatch,
		TotalBatches:               totalBatches,
		LastProcessedLinePreview:   linePreview,
	}

	// This is a background update, don't fail the import if it fails
	if err := i.storage.UpdateFileProgress(progressData); err != nil {
		// Log error but don't stop processing
		fmt.Printf("Warning: Failed to update progress for file %s: %v\n", fileID, err)
	}
}

// updateStreamingProgress updates progress for streaming imports (no total lines known)
func (i *Importer) updateStreamingProgress(fileID string, linesProcessed int64, startTime time.Time, lastRecord []string) {
	elapsed := time.Since(startTime)
	
	// Calculate processing rate (lines per second)
	processingRate := float64(linesProcessed) / elapsed.Seconds()
	
	// Get a preview of the current line being processed
	var linePreview string
	if lastRecord != nil && len(lastRecord) > 0 {
		// Show first few columns as preview
		preview := make([]string, 0, 3)
		for i, col := range lastRecord {
			if i >= 3 { // Only show first 3 columns
				break
			}
			preview = append(preview, col)
		}
		linePreview = strings.Join(preview, ", ")
		if len(linePreview) > 100 {
			linePreview = linePreview[:100] + "..."
		}
	}

	// Calculate progress percentage if we know total lines
	progressPercentage := 0
	var estimatedCompletion time.Time
	
	// Get the file record to check if we have total lines
	files, err := i.storage.ListMarketDataFiles()
	if err == nil {
		for _, f := range files {
			if f.ID == fileID && f.TotalLines > 0 {
				progressPercentage = int((linesProcessed * 100) / f.TotalLines)
				// Calculate estimated completion time
				if processingRate > 0 {
					remainingLines := f.TotalLines - linesProcessed
					remainingSeconds := float64(remainingLines) / processingRate
					estimatedCompletion = time.Now().Add(time.Duration(remainingSeconds) * time.Second)
				}
				break
			}
		}
	}

	// Update the database with streaming progress
	progressData := storage.ProgressUpdate{
		FileID:                     fileID,
		ProgressPercentage:         progressPercentage,
		LinesProcessed:             linesProcessed,
		ProcessingRate:             processingRate,
		EstimatedCompletionTime:    estimatedCompletion,
		CurrentBatch:               0,
		TotalBatches:               0,
		LastProcessedLinePreview:   linePreview,
	}

	// This is a background update, don't fail the import if it fails
	if err := i.storage.UpdateFileProgress(progressData); err != nil {
		// Log error but don't stop processing
		fmt.Printf("Warning: Failed to update streaming progress for file %s: %v\n", fileID, err)
	}
	
	log.Printf("Streaming progress: %d lines processed at %.1f lines/sec", linesProcessed, processingRate)
}

// updateStreamingProgressSafe is a safer version that handles errors gracefully
func (i *Importer) updateStreamingProgressSafe(fileID string, linesProcessed int64, startTime time.Time, lastRecord []string) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in updateStreamingProgressSafe: %v", r)
		}
	}()

	elapsed := time.Since(startTime)
	
	// Calculate processing rate (lines per second)
	processingRate := float64(linesProcessed) / elapsed.Seconds()
	
	// Get a preview of the current line being processed
	var linePreview string
	if lastRecord != nil && len(lastRecord) > 0 {
		// Show first few columns as preview
		preview := make([]string, 0, 3)
		for i, col := range lastRecord {
			if i >= 3 { // Only show first 3 columns
				break
			}
			preview = append(preview, col)
		}
		linePreview = strings.Join(preview, ", ")
		if len(linePreview) > 100 {
			linePreview = linePreview[:100] + "..."
		}
	}

	// Calculate progress percentage if we know total lines
	progressPercentage := 0
	var estimatedCompletion time.Time
	
	// Get the file record to check if we have total lines
	files, err := i.storage.ListMarketDataFiles()
	if err != nil {
		return fmt.Errorf("failed to get file list for progress calculation: %w", err)
	}
	
	for _, f := range files {
		if f.ID == fileID && f.TotalLines > 0 {
			progressPercentage = int((linesProcessed * 100) / f.TotalLines)
			// Calculate estimated completion time
			if processingRate > 0 {
				remainingLines := f.TotalLines - linesProcessed
				remainingSeconds := float64(remainingLines) / processingRate
				estimatedCompletion = time.Now().Add(time.Duration(remainingSeconds) * time.Second)
			}
			break
		}
	}

	// Update the database with streaming progress
	progressData := storage.ProgressUpdate{
		FileID:                     fileID,
		ProgressPercentage:         progressPercentage,
		LinesProcessed:             linesProcessed,
		ProcessingRate:             processingRate,
		EstimatedCompletionTime:    estimatedCompletion,
		CurrentBatch:               0,
		TotalBatches:               0,
		LastProcessedLinePreview:   linePreview,
	}

	// This is a background update, don't fail the import if it fails
	if err := i.storage.UpdateFileProgress(progressData); err != nil {
		return fmt.Errorf("failed to update progress in database: %w", err)
	}
	
	log.Printf("Streaming progress: %d lines processed at %.1f lines/sec (%d%%)", linesProcessed, processingRate, progressPercentage)
	return nil
}

// updateStreamingProgressCached updates progress using cached total lines to avoid DB queries
func (i *Importer) updateStreamingProgressCached(fileID string, linesProcessed, totalLines int64, startTime time.Time, lastRecord []string) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in updateStreamingProgressCached: %v", r)
		}
	}()

	elapsed := time.Since(startTime)
	
	// Calculate processing rate (lines per second)
	processingRate := float64(linesProcessed) / elapsed.Seconds()
	
	// Get a preview of the current line being processed
	var linePreview string
	if lastRecord != nil && len(lastRecord) > 0 {
		// Show first few columns as preview
		preview := make([]string, 0, 3)
		for i, col := range lastRecord {
			if i >= 3 { // Only show first 3 columns
				break
			}
			preview = append(preview, col)
		}
		linePreview = strings.Join(preview, ", ")
		if len(linePreview) > 100 {
			linePreview = linePreview[:100] + "..."
		}
	}

	// Calculate progress percentage using cached total lines
	progressPercentage := 0
	var estimatedCompletion time.Time
	
	if totalLines > 0 {
		progressPercentage = int((linesProcessed * 100) / totalLines)
		// Calculate estimated completion time
		if processingRate > 0 {
			remainingLines := totalLines - linesProcessed
			remainingSeconds := float64(remainingLines) / processingRate
			estimatedCompletion = time.Now().Add(time.Duration(remainingSeconds) * time.Second)
		}
	}

	// Update the database with streaming progress
	progressData := storage.ProgressUpdate{
		FileID:                     fileID,
		ProgressPercentage:         progressPercentage,
		LinesProcessed:             linesProcessed,
		ProcessingRate:             processingRate,
		EstimatedCompletionTime:    estimatedCompletion,
		CurrentBatch:               0,
		TotalBatches:               0,
		LastProcessedLinePreview:   linePreview,
	}

	// This is a background update, don't fail the import if it fails
	if err := i.storage.UpdateFileProgress(progressData); err != nil {
		return fmt.Errorf("failed to update progress in database: %w", err)
	}
	
	log.Printf("Streaming progress: %d lines processed at %.1f lines/sec (%d%%)", linesProcessed, processingRate, progressPercentage)
	return nil
}