package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/tradestax/traedor/pkg/storage"
	"github.com/tradestax/traedor/pkg/services"
	"github.com/tradestax/traedor/pkg/signals"
	"github.com/tradestax/traedor/pkg/types"
)

type Server struct {
	storage    storage.IStorage
	router     *mux.Router
	importPool *services.ImportPool
}

func NewServer(store storage.IStorage) *Server {
	s := &Server{
		storage:    store,
		router:     mux.NewRouter(),
		importPool: services.NewImportPool(store),
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
	api.HandleFunc("/runs/{id}/trades/stream", s.handleStreamTrades).Methods("GET", "OPTIONS")
	api.HandleFunc("/runs/{id}/signals", s.handleGetSignals).Methods("GET", "OPTIONS")
	
	// Signal definitions
	api.HandleFunc("/signals", s.handleListSignals).Methods("GET", "OPTIONS")
	api.HandleFunc("/signals/available", s.handleListAvailableSignals).Methods("GET", "OPTIONS")
	api.HandleFunc("/signals", s.handleCreateSignal).Methods("POST", "OPTIONS")
	api.HandleFunc("/signals/{id}", s.handleGetSignal).Methods("GET", "OPTIONS")
	api.HandleFunc("/signals/{id}", s.handleUpdateSignal).Methods("PUT", "OPTIONS")
	api.HandleFunc("/signals/{id}", s.handleDeleteSignal).Methods("DELETE", "OPTIONS")
	
	// Configuration
	api.HandleFunc("/config/symbols", s.handleGetSymbols).Methods("GET", "OPTIONS")
	api.HandleFunc("/config/symbols/availability", s.handleGetSymbolDataAvailability).Methods("GET", "OPTIONS")
	api.HandleFunc("/config/timeframes", s.handleGetTimeframes).Methods("GET", "OPTIONS")
	
	// Market data import
	api.HandleFunc("/data/files", s.handleListDataFiles).Methods("GET", "OPTIONS")
	api.HandleFunc("/data/files/failed", s.handleDeleteFailedImports).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/data/files/pending", s.handleDeletePendingImports).Methods("DELETE", "OPTIONS")
	api.HandleFunc("/data/files/stuck", s.handleCleanupStuckImports).Methods("POST", "OPTIONS")
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

func (s *Server) Shutdown() {
	if s.importPool != nil {
		s.importPool.Stop()
	}
}

// Run handlers
func (s *Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var config storage.RunConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		log.Printf("Failed to decode request body: %v", err)
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	
	log.Printf("Received create run request: %+v", config)
	
	// Validate the configuration
	if config.Symbol == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Symbol is required")
		return
	}
	
	// Validate that the symbol exists in the database
	symbolExists, err := s.storage.SymbolExists(config.Symbol)
	if err != nil {
		log.Printf("Error checking symbol existence: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to validate symbol")
		return
	}
	if !symbolExists {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Symbol '%s' not found in database", config.Symbol))
		return
	}
	
	// Handle signals with parameters (new format)
	var signalIDs []string
	if len(config.SignalsWithParams) > 0 {
		for _, signalWithParams := range config.SignalsWithParams {
			
			var baseSignalDef storage.SignalDefinition
			found := false
			
			// First, check if it's an existing signal definition ID
			signalDefinitions, err := s.storage.GetSignalDefinitions()
			if err != nil {
				writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get signal definitions: %v", err))
				return
			}
			
			for _, def := range signalDefinitions {
				if def.ID == signalWithParams.SignalDefinitionID || def.Name == signalWithParams.SignalDefinitionID {
					baseSignalDef = def
					found = true
					break
				}
			}
			
			// If not found, check if it's a signal type (for available signal generators)
			if !found {
				generator, exists := signals.GetSignalGenerator(signalWithParams.SignalDefinitionID)
				if exists {
					// Create a base signal definition from the signal type
					baseSignalDef = storage.SignalDefinition{
						Name:        generator.GetName(),
						Description: generator.GetDescription(),
						Type:        signalWithParams.SignalDefinitionID,
						Parameters:  generator.GetDefaultParameters(),
						Active:      true,
					}
					found = true
				}
			}
			
			if !found {
				writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Signal definition or type '%s' not found", signalWithParams.SignalDefinitionID))
				return
			}
			
			// Create signal instance with merged parameters
			mergedParams := make(map[string]interface{})
			for k, v := range baseSignalDef.Parameters {
				mergedParams[k] = v
			}
			for k, v := range signalWithParams.Parameters {
				mergedParams[k] = v
			}
			
			// Create a descriptive name based on parameters
			instanceName := baseSignalDef.Name
			if aggInterval, hasAgg := mergedParams["aggregation_interval"]; hasAgg {
				instanceName = fmt.Sprintf("%s_%vm", baseSignalDef.Name, aggInterval)
			}
			instanceName = fmt.Sprintf("%s_%d", instanceName, time.Now().UnixNano())
			
			signalInstance := storage.SignalDefinition{
				Name:        instanceName,
				Description: baseSignalDef.Description,
				Type:        baseSignalDef.Type,
				Parameters:  mergedParams,
				Active:      true,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			
			// Create the signal instance and get the actual ID assigned by the database
			log.Printf("Creating signal instance: %+v", signalInstance)
			if err := s.storage.CreateSignalDefinition(signalInstance); err != nil {
				log.Printf("Failed to create signal instance %s: %v", signalInstance.Name, err)
				writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create signal instance %s: %v", signalInstance.Name, err))
				return
			}
			
			// Get the actual ID that was assigned by the database
			signalDefinitions, err = s.storage.GetSignalDefinitions()
			if err != nil {
				writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to get updated signal definitions: %v", err))
				return
			}
			
			var actualInstanceID string
			for _, def := range signalDefinitions {
				if def.Name == instanceName {
					actualInstanceID = def.ID
					break
				}
			}
			
			if actualInstanceID == "" {
				writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to find created signal instance %s", instanceName))
				return
			}
			
			signalIDs = append(signalIDs, actualInstanceID)
		}
		
		// Use the created signal instance IDs
		config.Signals = signalIDs
	} else if len(config.SignalDefinitions) > 0 {
		// Legacy: Create signal definitions if provided (for backward compatibility)
		for _, signalDef := range config.SignalDefinitions {
			// Generate unique ID for signal definition
			signalDef.ID = fmt.Sprintf("%s_%d", signalDef.Name, time.Now().UnixNano())
			signalDef.CreatedAt = time.Now()
			signalDef.UpdatedAt = time.Now()
			
			// Create the signal definition
			if err := s.storage.CreateSignalDefinition(signalDef); err != nil {
				writeErrorResponse(w, http.StatusInternalServerError, fmt.Sprintf("Failed to create signal definition %s: %v", signalDef.Name, err))
				return
			}
			
			signalIDs = append(signalIDs, signalDef.ID)
		}
		
		// Use the created signal IDs
		config.Signals = signalIDs
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
	
	// Parse query parameters for pagination
	limitStr := r.URL.Query().Get("limit")
	offsetStr := r.URL.Query().Get("offset")
	
	limit := 100 // Default limit
	offset := 0  // Default offset
	
	if limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil && parsedLimit > 0 {
			limit = parsedLimit
		}
	}
	
	if offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil && parsedOffset >= 0 {
			offset = parsedOffset
		}
	}
	
	// Get paginated trades
	trades, total, err := s.storage.GetTradesPaginated(runID, limit, offset)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get trades")
		return
	}
	
	// Create response in format expected by frontend
	response := map[string]interface{}{
		"trades": trades,
		"pagination": map[string]interface{}{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	}
	
	writeJSONResponse(w, http.StatusOK, response)
}

func (s *Server) handleStreamTrades(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["id"]
	
	log.Printf("Streaming all trades for run %s", runID)
	
	// Set headers for JSON streaming
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	
	// Start JSON array
	w.Write([]byte("{\"trades\":["))
	
	first := true
	totalCount := 0
	
	// Stream trades using the callback approach
	err := s.storage.StreamTrades(runID, func(trade *types.Trade) error {
		totalCount++
		
		// Add comma separator for all trades except the first
		if !first {
			if _, err := w.Write([]byte(",")); err != nil {
				return fmt.Errorf("failed to write separator: %w", err)
			}
		}
		first = false
		
		// Marshal and write each trade
		tradeJSON, err := json.Marshal(trade)
		if err != nil {
			return fmt.Errorf("failed to marshal trade: %w", err)
		}
		
		if _, err := w.Write(tradeJSON); err != nil {
			return fmt.Errorf("failed to write trade: %w", err)
		}
		
		// Flush data to client immediately (streaming)
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		
		return nil
	})
	
	if err != nil {
		log.Printf("Error streaming trades: %v", err)
		// If we haven't written any trades yet, we can still write an error response
		if first {
			w.Write([]byte("]}"))
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to stream trades")
			return
		}
		// If we've already started streaming, just close the array
		w.Write([]byte("]}"))
		return
	}
	
	// Close JSON array and add metadata
	metadata := fmt.Sprintf("],\"total\":%d}", totalCount)
	w.Write([]byte(metadata))
	
	log.Printf("Successfully streamed %d trades for run %s", totalCount, runID)
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

func (s *Server) handleListAvailableSignals(w http.ResponseWriter, r *http.Request) {
	// Get available signal types from the signal generators
	availableTypes := signals.GetAvailableSignalGenerators()
	result := make([]map[string]interface{}, len(availableTypes))

	for i, signalType := range availableTypes {
		generator, exists := signals.GetSignalGenerator(signalType)
		if !exists {
			continue
		}
		
		result[i] = map[string]interface{}{
			"id":          signalType,
			"name":        generator.GetName(),
			"type":        signalType,
			"description": generator.GetDescription(),
			"parameters":  generator.GetDefaultParameters(),
		}
	}

	writeJSONResponse(w, http.StatusOK, result)
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
	// Get symbol details from database
	symbols, err := s.storage.GetSymbolDetails()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get symbols")
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(symbols)
}

func (s *Server) handleGetTimeframes(w http.ResponseWriter, r *http.Request) {
	// Get timeframe details from database
	timeframes, err := s.storage.GetTimeframeDetails()
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get timeframes")
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(timeframes)
}

func (s *Server) handleGetSymbolDataAvailability(w http.ResponseWriter, r *http.Request) {
	symbol := r.URL.Query().Get("symbol")
	if symbol == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Symbol parameter is required")
		return
	}
	
	log.Printf("Checking data availability for symbol: %s", symbol)
	
	availability, err := s.storage.GetSymbolDataAvailability(symbol)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get data availability")
		return
	}
	
	if availability == nil {
		writeErrorResponse(w, http.StatusNotFound, fmt.Sprintf("No market data available for symbol %s. Please ensure data has been imported for this symbol.", symbol))
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(availability)
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