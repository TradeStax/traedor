package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/tradestax/traedor/pkg/storage"
)

type Server struct {
	storage storage.IStorage
	router  *mux.Router
}

func NewServer(store storage.IStorage) *Server {
	s := &Server{
		storage: store,
		router:  mux.NewRouter(),
	}
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	api := s.router.PathPrefix("/api").Subrouter()
	
	// Enable CORS middleware on the API subrouter
	api.Use(corsMiddleware)
	
	// Run management
	api.HandleFunc("/runs", s.handleCreateRun).Methods("POST", "OPTIONS")
	api.HandleFunc("/runs", s.handleListRuns).Methods("GET", "OPTIONS")
	api.HandleFunc("/runs/{id}", s.handleGetRun).Methods("GET", "OPTIONS")
	api.HandleFunc("/runs/{id}/cancel", s.handleCancelRun).Methods("POST", "OPTIONS")
	api.HandleFunc("/runs/{id}/retry", s.handleRetryRun).Methods("POST", "OPTIONS")
	
	// Run data
	api.HandleFunc("/runs/{id}/trades", s.handleGetTrades).Methods("GET", "OPTIONS")
	api.HandleFunc("/runs/{id}/signals", s.handleGetSignals).Methods("GET", "OPTIONS")
	
	// Signal definitions
	api.HandleFunc("/signals", s.handleListSignals).Methods("GET", "OPTIONS")
	api.HandleFunc("/signals", s.handleCreateSignal).Methods("POST", "OPTIONS")
	api.HandleFunc("/signals/{id}", s.handleGetSignal).Methods("GET", "OPTIONS")
	api.HandleFunc("/signals/{id}", s.handleUpdateSignal).Methods("PUT", "OPTIONS")
	api.HandleFunc("/signals/{id}", s.handleDeleteSignal).Methods("DELETE", "OPTIONS")
	
	// Configuration
	api.HandleFunc("/config/symbols", s.handleGetSymbols).Methods("GET", "OPTIONS")
	api.HandleFunc("/config/timeframes", s.handleGetTimeframes).Methods("GET", "OPTIONS")
	
	// Market data import
	api.HandleFunc("/data/files", s.handleListDataFiles).Methods("GET", "OPTIONS")
	api.HandleFunc("/data/files/failed", s.handleDeleteFailedImports).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/data/files/pending", s.handleDeletePendingImports).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/data/files/{id}", s.handleDeleteDataFile).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/data/files/{id}/retry", s.handleRetryDataFile).Methods("POST", "OPTIONS")
	api.HandleFunc("/data/scan", s.handleScanDataFiles).Methods("POST", "OPTIONS")
	api.HandleFunc("/data/import/{id}", s.handleImportDataFile).Methods("POST", "OPTIONS")
	api.HandleFunc("/data/ohlc", s.handleGetOHLCData).Methods("GET", "OPTIONS")
	
	// Health check
	api.HandleFunc("/health", s.handleHealth).Methods("GET", "OPTIONS")
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Run handlers
func (s *Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var config storage.RunConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	// Validate the configuration
	if config.Symbol == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Symbol is required")
		return
	}
	
	// Create the run (initially in pending status)
	run, err := s.storage.CreateRun(config)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create run")
		return
	}
	
	// Immediately queue the run for processing
	if err := s.storage.UpdateRunStatus(run.ID, storage.RunStatusQueued, nil); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to queue run")
		return
	}
	
	// Update the run status for response
	run.Status = storage.RunStatusQueued
	run.StatusMessage = "Run queued for processing"
	
	writeJSONResponse(w, http.StatusCreated, run)
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	filter := storage.RunFilter{}
	
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = storage.RunStatus(status)
	}
	
	if symbol := r.URL.Query().Get("symbol"); symbol != "" {
		filter.Symbol = symbol
	}
	
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}
	
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}
	
	runs, err := s.storage.ListRuns(filter)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to list runs")
		return
	}
	
	writeJSONResponse(w, http.StatusOK, runs)
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["id"]
	
	run, err := s.storage.GetRun(runID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, "Run not found")
		return
	}
	
	writeJSONResponse(w, http.StatusOK, run)
}

func (s *Server) handleCancelRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["id"]
	
	if err := s.storage.CancelRun(runID); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to cancel run")
		return
	}
	
	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "Run cancelled"})
}

func (s *Server) handleRetryRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["id"]
	
	if err := s.storage.RetryFailedRun(runID); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to retry run")
		return
	}
	
	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "Run queued for retry"})
}

func (s *Server) handleGetTrades(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["id"]
	
	trades, err := s.storage.GetTrades(runID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get trades")
		return
	}
	
	writeJSONResponse(w, http.StatusOK, trades)
}

func (s *Server) handleGetSignals(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["id"]
	
	signals, err := s.storage.GetSignals(runID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get signals")
		return
	}
	
	writeJSONResponse(w, http.StatusOK, signals)
}

// Signal definition handlers (simplified for now)
func (s *Server) handleListSignals(w http.ResponseWriter, r *http.Request) {
	signals, err := s.storage.GetSignalDefinitions()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to list signals")
		return
	}
	
	writeJSONResponse(w, http.StatusOK, signals)
}

func (s *Server) handleCreateSignal(w http.ResponseWriter, r *http.Request) {
	var signal storage.SignalDefinition
	if err := json.NewDecoder(r.Body).Decode(&signal); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	if err := s.storage.CreateSignalDefinition(signal); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create signal")
		return
	}
	
	writeJSONResponse(w, http.StatusCreated, signal)
}

func (s *Server) handleGetSignal(w http.ResponseWriter, r *http.Request) {
	writeErrorResponse(w, http.StatusNotImplemented, "Not implemented")
}

func (s *Server) handleUpdateSignal(w http.ResponseWriter, r *http.Request) {
	writeErrorResponse(w, http.StatusNotImplemented, "Not implemented")
}

func (s *Server) handleDeleteSignal(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	signalID := vars["id"]
	
	if err := s.storage.DeleteSignalDefinition(signalID); err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete signal")
		return
	}
	
	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "Signal deleted"})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
		"service":   "traedor-api",
	}
	writeJSONResponse(w, http.StatusOK, health)
}

func (s *Server) handleOptions(w http.ResponseWriter, r *http.Request) {
	// This is handled by the CORS middleware
	w.WriteHeader(http.StatusOK)
}

// Helper functions
// handleDeleteDataFile deletes a specific market data file by ID
func (s *Server) handleDeleteDataFile(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	fileID := vars["id"]
	
	if fileID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "File ID is required")
		return
	}
	
	// Delete the file record
	err := s.storage.DeleteMarketDataFile(fileID)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete file")
		return
	}
	
	writeJSONResponse(w, http.StatusOK, map[string]string{
		"message": "File deleted successfully",
	})
}

// handleDeleteFailedImports deletes all failed import records
func (s *Server) handleDeleteFailedImports(w http.ResponseWriter, r *http.Request) {
	count, err := s.storage.DeleteFailedImports()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete failed imports")
		return
	}
	
	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Failed imports deleted successfully",
		"count":   count,
	})
}

// handleDeletePendingImports deletes all pending import records
func (s *Server) handleDeletePendingImports(w http.ResponseWriter, r *http.Request) {
	count, err := s.storage.DeletePendingImports()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to delete pending imports")
		return
	}
	
	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Pending imports deleted successfully",
		"count":   count,
	})
}

func writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func (s *Server) handleGetSymbols(w http.ResponseWriter, r *http.Request) {
	availableSymbols, err := s.storage.GetAvailableSymbols()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get available symbols")
		return
	}
	
	// Map symbols to descriptions
	symbolDescriptions := map[string]string{
		"/MES": "Micro E-mini S&P 500",
		"/MNQ": "Micro E-mini NASDAQ-100", 
		"/MYM": "Micro E-mini Dow",
		"/M2K": "Micro E-mini Russell 2000",
		"/ES":  "E-mini S&P 500",
		"/NQ":  "E-mini NASDAQ-100",
	}
	
	symbols := make([]map[string]string, 0, len(availableSymbols))
	for _, symbol := range availableSymbols {
		description := symbolDescriptions[symbol]
		if description == "" {
			description = symbol // Fallback to symbol name if no description
		}
		symbols = append(symbols, map[string]string{
			"name":        symbol,
			"description": description,
		})
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(symbols)
}

func (s *Server) handleGetTimeframes(w http.ResponseWriter, r *http.Request) {
	availableTimeframes, err := s.storage.GetAvailableTimeframes()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get available timeframes")
		return
	}
	
	// Map timeframes to descriptions
	timeframeDescriptions := map[string]string{
		"1m":  "1 minute",
		"5m":  "5 minutes", 
		"15m": "15 minutes",
		"30m": "30 minutes",
		"1h":  "1 hour",
		"4h":  "4 hours",
		"1d":  "1 day",
	}
	
	timeframes := make([]map[string]string, 0, len(availableTimeframes))
	for _, tf := range availableTimeframes {
		description := timeframeDescriptions[tf]
		if description == "" {
			description = tf // Fallback to timeframe value if no description
		}
		timeframes = append(timeframes, map[string]string{
			"value":       tf,
			"description": description,
		})
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(timeframes)
}

func writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}