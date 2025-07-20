package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/tradestax/traedor/internal/config"
	"github.com/tradestax/traedor/pkg/signals"
	"github.com/tradestax/traedor/pkg/storage"
)

type Server struct {
	router         *mux.Router
	storage        storage.IStorage
	config         *config.Config
	server         *http.Server
	runnerCh       chan RunRequest
	signalWorkflow *signals.SignalWorkflow
}

type RunRequest struct {
	Config   storage.RunConfig `json:"config"`
	Response chan RunResponse
}

type RunResponse struct {
	RunID string `json:"run_id"`
	Error error  `json:"error,omitempty"`
}

func NewServer(cfg *config.Config, store storage.IStorage) *Server {
	s := &Server{
		router:         mux.NewRouter(),
		storage:        store,
		config:         cfg,
		runnerCh:       make(chan RunRequest, 10),
		signalWorkflow: signals.NewSignalWorkflow(store),
	}

	s.setupRoutes()
	
	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.API.Host, cfg.API.Port),
		Handler:      s.corsMiddleware(s.router),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s
}

func (s *Server) setupRoutes() {
	// API v1 routes
	api := s.router.PathPrefix("/api/v1").Subrouter()
	
	// Health check
	api.HandleFunc("/health", s.handleHealth).Methods("GET")
	
	// Run management
	api.HandleFunc("/runs", s.handleListRuns).Methods("GET")
	api.HandleFunc("/runs", s.handleCreateRun).Methods("POST")
	api.HandleFunc("/runs/{id}", s.handleGetRun).Methods("GET")
	api.HandleFunc("/runs/{id}/trades", s.handleGetRunTrades).Methods("GET")
	api.HandleFunc("/runs/{id}/signals", s.handleGetRunSignals).Methods("GET")
	
	// Signal definitions
	api.HandleFunc("/signals", s.handleListSignalDefinitions).Methods("GET")
	api.HandleFunc("/signals", s.handleCreateSignalDefinition).Methods("POST")
	api.HandleFunc("/signals/{id}", s.handleUpdateSignalDefinition).Methods("PUT")
	api.HandleFunc("/signals/{id}", s.handleDeleteSignalDefinition).Methods("DELETE")
	api.HandleFunc("/signals/available", s.handleListAvailableSignals).Methods("GET")
	api.HandleFunc("/signals/validate", s.handleValidateSignal).Methods("POST")
	api.HandleFunc("/signals/compatibility", s.handleCheckCompatibility).Methods("POST")
	
	// Configuration
	api.HandleFunc("/config/symbols", s.handleListSymbols).Methods("GET")
	api.HandleFunc("/config/timeframes", s.handleListTimeframes).Methods("GET")
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
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

func (s *Server) Start() error {
	fmt.Printf("Starting API server on %s\n", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *Server) GetRunnerChannel() chan RunRequest {
	return s.runnerCh
}

// Handlers

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.respondJSON(w, http.StatusOK, map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (s *Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	filter := storage.RunFilter{
		Limit:  50,
		Offset: 0,
	}

	// Parse query parameters
	if symbol := r.URL.Query().Get("symbol"); symbol != "" {
		filter.Symbol = symbol
	}
	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = storage.RunStatus(status)
	}

	runs, err := s.storage.ListRuns(filter)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, runs)
}

func (s *Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var runConfig storage.RunConfig
	if err := json.NewDecoder(r.Body).Decode(&runConfig); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Send to runner channel
	responseCh := make(chan RunResponse)
	s.runnerCh <- RunRequest{
		Config:   runConfig,
		Response: responseCh,
	}

	// Wait for response
	response := <-responseCh
	if response.Error != nil {
		s.respondError(w, http.StatusInternalServerError, response.Error.Error())
		return
	}

	s.respondJSON(w, http.StatusCreated, map[string]string{
		"run_id": response.RunID,
		"status": "started",
	})
}

func (s *Server) handleGetRun(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["id"]

	run, err := s.storage.GetRun(runID)
	if err != nil {
		s.respondError(w, http.StatusNotFound, "Run not found")
		return
	}

	s.respondJSON(w, http.StatusOK, run)
}

func (s *Server) handleGetRunTrades(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["id"]

	trades, err := s.storage.GetTrades(runID)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, trades)
}

func (s *Server) handleGetRunSignals(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	runID := vars["id"]

	signals, err := s.storage.GetSignals(runID)
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, signals)
}

func (s *Server) handleListSignalDefinitions(w http.ResponseWriter, r *http.Request) {
	definitions, err := s.storage.GetSignalDefinitions()
	if err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, definitions)
}

func (s *Server) handleCreateSignalDefinition(w http.ResponseWriter, r *http.Request) {
	var def storage.SignalDefinition
	if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.storage.CreateSignalDefinition(def); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusCreated, def)
}

func (s *Server) handleUpdateSignalDefinition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	var def storage.SignalDefinition
	if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := s.storage.UpdateSignalDefinition(id, def); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusOK, def)
}

func (s *Server) handleDeleteSignalDefinition(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	if err := s.storage.DeleteSignalDefinition(id); err != nil {
		s.respondError(w, http.StatusInternalServerError, err.Error())
		return
	}

	s.respondJSON(w, http.StatusNoContent, nil)
}

func (s *Server) handleListAvailableSignals(w http.ResponseWriter, r *http.Request) {
	signals := s.getAvailableSignals()
	s.respondJSON(w, http.StatusOK, signals)
}

func (s *Server) handleValidateSignal(w http.ResponseWriter, r *http.Request) {
	var def storage.SignalDefinition
	if err := json.NewDecoder(r.Body).Decode(&def); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	validator := signals.NewSignalValidator()
	if err := validator.ValidateSignalDefinition(def); err != nil {
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"valid": true,
	})
}

func (s *Server) handleCheckCompatibility(w http.ResponseWriter, r *http.Request) {
	var request struct {
		SignalIDs []string `json:"signal_ids"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		s.respondError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	if err := signals.ValidateSignalCompatibility(request.SignalIDs, s.storage); err != nil {
		s.respondJSON(w, http.StatusOK, map[string]interface{}{
			"compatible": false,
			"error":      err.Error(),
		})
		return
	}

	s.respondJSON(w, http.StatusOK, map[string]interface{}{
		"compatible": true,
	})
}

func (s *Server) handleListSymbols(w http.ResponseWriter, r *http.Request) {
	// For now, return hardcoded symbols. Later can be fetched from database
	symbols := []map[string]interface{}{
		{"name": "/MES", "description": "Micro E-mini S&P 500"},
		{"name": "/MNQ", "description": "Micro E-mini Nasdaq-100"},
		{"name": "/MYM", "description": "Micro E-mini Dow"},
		{"name": "/M2K", "description": "Micro E-mini Russell 2000"},
		{"name": "/ES", "description": "E-mini S&P 500"},
		{"name": "/NQ", "description": "E-mini Nasdaq-100"},
	}
	s.respondJSON(w, http.StatusOK, symbols)
}

func (s *Server) handleListTimeframes(w http.ResponseWriter, r *http.Request) {
	timeframes := []map[string]interface{}{
		{"value": "1m", "description": "1 Minute"},
		{"value": "5m", "description": "5 Minutes"},
		{"value": "15m", "description": "15 Minutes"},
		{"value": "30m", "description": "30 Minutes"},
		{"value": "1h", "description": "1 Hour"},
		{"value": "4h", "description": "4 Hours"},
		{"value": "1d", "description": "1 Day"},
	}
	s.respondJSON(w, http.StatusOK, timeframes)
}

// Helper methods

func (s *Server) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if data != nil {
		json.NewEncoder(w).Encode(data)
	}
}

func (s *Server) respondError(w http.ResponseWriter, status int, message string) {
	s.respondJSON(w, status, map[string]string{"error": message})
}

func (s *Server) getAvailableSignals() []map[string]interface{} {
	// Import signals package to ensure signal generators are registered
	_ = s.storage
	
	signals := []map[string]interface{}{
		{
			"name":        "sma_crossover",
			"description": "Simple Moving Average Crossover",
			"parameters": map[string]interface{}{
				"short_period": 20,
				"long_period":  50,
			},
		},
		{
			"name":        "rsi",
			"description": "Relative Strength Index",
			"parameters": map[string]interface{}{
				"period":           14,
				"overbought_level": 70.0,
				"oversold_level":   30.0,
			},
		},
	}
	
	return signals
}