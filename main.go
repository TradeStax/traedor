package main

import (
	"flag"
	"log"
	"os"

	"github.com/tradestax/traedor/internal/config"
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
	log.Println("API mode is deprecated. Please use cmd/server/main.go instead.")
	log.Println("Run: go run cmd/server/main.go")
	os.Exit(1)
}
