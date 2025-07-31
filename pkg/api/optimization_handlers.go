package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/tradestax/traedor/pkg/optimization"
	"github.com/tradestax/traedor/pkg/storage"
)

// handleCreateOptimization creates a new signal optimization
func (s *Server) handleCreateOptimization(w http.ResponseWriter, r *http.Request) {
	var config storage.OptimizationConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		log.Printf("Failed to decode optimization request body: %v", err)
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	log.Printf("Received create optimization request: %+v", config)

	// Validate the configuration
	if config.Name == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Optimization name is required")
		return
	}

	if len(config.ParameterRanges) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "At least one parameter range is required")
		return
	}

	// Validate that the symbol exists in the database
	symbolExists, err := s.storage.SymbolExists(config.BaseRunConfig.Symbol)
	if err != nil {
		log.Printf("Error checking symbol existence: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to validate symbol")
		return
	}
	if !symbolExists {
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Symbol '%s' not found in database", config.BaseRunConfig.Symbol))
		return
	}

	// Create optimizer to generate parameter combinations
	optimizer := optimization.NewOptimizer(s.storage)
	
	// Generate parameter combinations
	parameterCombinations, err := optimizer.GenerateParameterCombinations(config)
	if err != nil {
		log.Printf("Failed to generate parameter combinations: %v", err)
		writeErrorResponse(w, http.StatusBadRequest, fmt.Sprintf("Failed to generate parameter combinations: %v", err))
		return
	}

	log.Printf("Generated %d parameter combinations", len(parameterCombinations))

	// Create the optimization (initially in pending status)
	optimization, err := s.storage.CreateOptimization(config)
	if err != nil {
		log.Printf("Failed to create optimization: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create optimization")
		return
	}

	// Update with total permutations and parameter sequence
	optimization.TotalPermutations = len(parameterCombinations)
	optimization.ParameterSequence = parameterCombinations

	// Store the parameter sequence in the database
	if err := s.storage.UpdateOptimizationSequence(optimization.ID, len(parameterCombinations), parameterCombinations); err != nil {
		log.Printf("Failed to update optimization with parameter sequence: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to update optimization sequence")
		return
	}

	// Queue the optimization for processing
	if err := s.storage.UpdateOptimizationStatus(optimization.ID, storage.OptimizationStatusQueued, 0, "Optimization queued for processing"); err != nil {
		log.Printf("Failed to queue optimization: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to queue optimization")
		return
	}

	// Update the optimization status for response
	optimization.Status = storage.OptimizationStatusQueued
	optimization.StatusMessage = "Optimization queued for processing"

	writeJSONResponse(w, http.StatusCreated, optimization)
}

// handleListOptimizations lists all optimizations with filtering
func (s *Server) handleListOptimizations(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters for filtering
	filter := storage.OptimizationFilter{}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = storage.OptimizationStatus(status)
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

	optimizations, err := s.storage.ListOptimizations(filter)
	if err != nil {
		log.Printf("Failed to list optimizations: %v", err)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to list optimizations")
		return
	}

	writeJSONResponse(w, http.StatusOK, optimizations)
}

// handleGetOptimization retrieves a specific optimization
func (s *Server) handleGetOptimization(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	optimizationID := vars["id"]

	optimization, err := s.storage.GetOptimization(optimizationID)
	if err != nil {
		log.Printf("Failed to get optimization %s: %v", optimizationID, err)
		writeErrorResponse(w, http.StatusNotFound, "Optimization not found")
		return
	}

	writeJSONResponse(w, http.StatusOK, optimization)
}

// handleCancelOptimization cancels an optimization
func (s *Server) handleCancelOptimization(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	optimizationID := vars["id"]

	if err := s.storage.CancelOptimization(optimizationID); err != nil {
		log.Printf("Failed to cancel optimization %s: %v", optimizationID, err)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to cancel optimization")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "Optimization cancelled"})
}

// handlePauseOptimization pauses an optimization
func (s *Server) handlePauseOptimization(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	optimizationID := vars["id"]

	if err := s.storage.PauseOptimization(optimizationID); err != nil {
		log.Printf("Failed to pause optimization %s: %v", optimizationID, err)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to pause optimization")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "Optimization paused"})
}

// handleResumeOptimization resumes a paused optimization
func (s *Server) handleResumeOptimization(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	optimizationID := vars["id"]

	if err := s.storage.ResumeOptimization(optimizationID); err != nil {
		log.Printf("Failed to resume optimization %s: %v", optimizationID, err)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to resume optimization")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]string{"message": "Optimization resumed"})
}

// handleCleanupOptimizationDuplicates cleans up duplicate results for an optimization
func (s *Server) handleCleanupOptimizationDuplicates(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	optimizationID := vars["id"]

	duplicatesRemoved, err := s.storage.CleanupDuplicateOptimizationRunResults(optimizationID)
	if err != nil {
		log.Printf("Failed to cleanup duplicates for optimization %s: %v", optimizationID, err)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to cleanup duplicate results")
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Duplicate results cleaned up",
		"duplicates_removed": duplicatesRemoved,
	})
}

// handleGetOptimizationResults retrieves results for an optimization
func (s *Server) handleGetOptimizationResults(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	optimizationID := vars["id"]

	results, err := s.storage.GetOptimizationRunResults(optimizationID)
	if err != nil {
		log.Printf("Failed to get optimization results for %s: %v", optimizationID, err)
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get optimization results")
		return
	}

	writeJSONResponse(w, http.StatusOK, results)
}