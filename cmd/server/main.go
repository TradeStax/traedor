package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/tradestax/traedor/internal/config"
	"github.com/tradestax/traedor/pkg/api"
	"github.com/tradestax/traedor/pkg/jobs"
	"github.com/tradestax/traedor/pkg/storage"
)

func main() {
	// Load configuration
	cfg := config.New()
	
	// Initialize storage
	store, err := storage.NewPostgresStorage(cfg.Database.ConnectionString)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()
	
	// Create API server
	apiServer := api.NewServer(store)
	
	// Create worker pool
	workerCount := getEnvInt("WORKER_COUNT", 2)
	workerPool := jobs.NewWorkerPool(cfg, store, workerCount)
	
	// Start worker pool
	workerPool.Start()
	defer workerPool.Stop()
	
	// Start HTTP server
	server := &http.Server{
		Addr:    ":" + strconv.Itoa(cfg.API.Port),
		Handler: apiServer,
	}
	
	// Start server in a goroutine
	go func() {
		log.Printf("Starting API server on port %d", cfg.API.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()
	
	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("Shutting down server...")
	
	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}
	
	log.Println("Server exited")
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}