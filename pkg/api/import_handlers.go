package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/tradestax/traedor/pkg/importer"
	"github.com/tradestax/traedor/pkg/storage"
)

// handleListDataFiles returns all imported data files
func (s *Server) handleListDataFiles(w http.ResponseWriter, r *http.Request) {
	files, err := s.storage.ListMarketDataFiles()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to list data files")
		return
	}
	
	writeJSONResponse(w, http.StatusOK, files)
}

// handleScanDataFiles triggers a manual scan of the data directory
func (s *Server) handleScanDataFiles(w http.ResponseWriter, r *http.Request) {
	// Just return the current status - the file watcher handles automatic scanning
	files, err := s.storage.ListMarketDataFiles()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get file status")
		return
	}
	
	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "File watcher running - imports are automatic",
		"files":   files,
	})
}

// handleImportDataFile imports a specific data file
func (s *Server) handleImportDataFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]
	
	// For new imports, fileID will be "new" and we need the file path
	var req struct {
		FilePath string `json:"file_path"`
	}
	
	if fileID == "new" {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
			return
		}
		
		if req.FilePath == "" {
			writeErrorResponse(w, http.StatusBadRequest, "File path is required")
			return
		}
		
		// Validate file exists
		if _, err := os.Stat(req.FilePath); err != nil {
			writeErrorResponse(w, http.StatusBadRequest, "File not found")
			return
		}
		
		// Start import asynchronously
		go func() {
			imp := importer.NewImporter(s.storage)
			// Use background context for long-running imports, not HTTP request context
			ctx := context.Background()
			if err := imp.ImportFile(ctx, req.FilePath); err != nil {
				// Error is logged in the file status
				fmt.Printf("Import failed for %s: %v\n", req.FilePath, err)
			}
		}()
		
		writeJSONResponse(w, http.StatusAccepted, map[string]string{
			"message": "Import started",
			"file":    req.FilePath,
		})
	} else {
		// Re-import existing file
		writeErrorResponse(w, http.StatusNotImplemented, "Re-import not yet implemented")
	}
}

// handleGetOHLCData returns OHLC data for a symbol
func (s *Server) handleGetOHLCData(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Symbol is required")
		return
	}
	
	// Parse time range
	startStr := r.URL.Query().Get("start")
	endStr := r.URL.Query().Get("end")
	
	startTime := time.Now().AddDate(0, -1, 0) // Default: 1 month ago
	endTime := time.Now()
	
	if startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			startTime = t
		}
	}
	
	if endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			endTime = t
		}
	}
	
	// Get data
	data, err := s.storage.GetOHLCData(symbol, startTime, endTime)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get OHLC data")
		return
	}
	
	// Format response for charting
	response := map[string]interface{}{
		"symbol": symbol,
		"start":  startTime,
		"end":    endTime,
		"data":   data,
	}
	
	writeJSONResponse(w, http.StatusOK, response)
}

// Helper types and functions
type FileInfo struct {
	Path    string
	Name    string
	Size    int64
	ModTime time.Time
}

func scanDataDirectory(dir string) ([]FileInfo, error) {
	var files []FileInfo
	
	// Read directory
	entries, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		
		// Check if it's a data file
		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if ext != ".txt" && ext != ".csv" {
			continue
		}
		
		files = append(files, FileInfo{
			Path:    filepath.Join(dir, entry.Name()),
			Name:    entry.Name(),
			Size:    entry.Size(),
			ModTime: entry.ModTime(),
		})
	}
	
	return files, nil
}

// handleRetryDataFile retries a failed import
func (s *Server) handleRetryDataFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]
	
	if fileID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "File ID is required")
		return
	}
	
	// Get the file record to check if it exists and is failed
	files, err := s.storage.ListMarketDataFiles()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to list files")
		return
	}
	
	var targetFile *storage.MarketDataFile
	for _, file := range files {
		if file.ID == fileID {
			targetFile = file
			break
		}
	}
	
	if targetFile == nil {
		writeErrorResponse(w, http.StatusNotFound, "File not found")
		return
	}
	
	if targetFile.Status != "failed" {
		writeErrorResponse(w, http.StatusBadRequest, "Only failed imports can be retried")
		return
	}
	
	// Reset the file status to pending to allow retry
	err = s.storage.UpdateMarketDataFileStatus(fileID, "pending", "Retry requested by user")
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to update file status for retry")
		return
	}
	
	// Trigger import by creating a new importer instance
	imp := importer.NewImporter(s.storage)
	go func() {
		// Use background context for long-running imports, not HTTP request context
		ctx := context.Background()
		if err := imp.ImportFile(ctx, targetFile.FilePath); err != nil {
			fmt.Printf("Retry import failed for %s: %v\n", targetFile.FilePath, err)
		}
	}()
	
	writeJSONResponse(w, http.StatusOK, map[string]string{
		"message": "Import retry started successfully",
	})
}
