package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tradestax/traedor/internal/config"
	"github.com/tradestax/traedor/pkg/api"
	"github.com/tradestax/traedor/pkg/storage"
	"github.com/tradestax/traedor/pkg/trader"
)

func main() {
	var (
		apiMode = flag.Bool("api", false, "Run in API mode")
		help    = flag.Bool("help", false, "Show help")
	)
	
	flag.Parse()
	
	if *help {
		flag.Usage()
		os.Exit(0)
	}
	
	cfg := config.New()
	
	if *apiMode {
		runAPIMode(cfg)
	} else {
		runCLIMode(cfg)
	}
}

func runCLIMode(cfg *config.Config) {
	trader := trader.NewTrader(cfg)
	trader.Run()
	trader.Summary()
}

func runAPIMode(cfg *config.Config) {
	// Initialize storage
	store, err := storage.NewPostgresStorage(cfg.Database.ConnectionString)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()
	
	// Create API server
	server := api.NewServer(cfg, store)
	
	// Create run manager
	runManager := api.NewRunManager(cfg, store, server.GetRunnerChannel())
	
	// Start run manager in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go runManager.Start(ctx)
	
	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	
	// Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()
	
	// Wait for shutdown signal
	<-sigChan
	
	fmt.Println("\nShutting down server...")
	
	// Shutdown server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
	
	fmt.Println("Server stopped")
}
