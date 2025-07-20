package jobs

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tradestax/traedor/internal/config"
	"github.com/tradestax/traedor/pkg/storage"
	"github.com/tradestax/traedor/pkg/trader"
)

type ProgressTrackingTrader struct {
	*trader.Trader
	storage           storage.IStorage
	runID             string
	totalTicks        int64
	processedTicks    int64
	lastProgressUpdate time.Time
	progressMutex     sync.Mutex
}

func NewProgressTrackingTrader(cfg *config.Config, store storage.IStorage, runID string) *ProgressTrackingTrader {
	baseTrader := trader.NewTraderWithStorage(cfg, store, runID)
	
	return &ProgressTrackingTrader{
		Trader:            baseTrader,
		storage:           store,
		runID:             runID,
		lastProgressUpdate: time.Now(),
	}
}

func (pt *ProgressTrackingTrader) Run() error {
	// Estimate total ticks based on time range and interval
	// This is a rough estimate - in a real implementation you might
	// scan the data files first to get accurate counts
	pt.estimateTotalTicks()
	
	// Start progress tracking goroutine
	stopProgress := make(chan bool)
	go pt.trackProgress(stopProgress)
	defer func() { stopProgress <- true }()
	
	// Run the actual trader
	return pt.Trader.Run()
}

func (pt *ProgressTrackingTrader) estimateTotalTicks() {
	// This is a simplified estimation
	// In reality, you'd want to scan data files or query the database
	pt.totalTicks = 10000 // Placeholder - should be calculated based on data range
}

func (pt *ProgressTrackingTrader) trackProgress(stop chan bool) {
	ticker := time.NewTicker(10 * time.Second) // Update progress every 10 seconds
	defer ticker.Stop()
	
	for {
		select {
		case <-stop:
			return
		case <-ticker.C:
			pt.updateProgress()
		}
	}
}

func (pt *ProgressTrackingTrader) updateProgress() {
	pt.progressMutex.Lock()
	defer pt.progressMutex.Unlock()
	
	if pt.totalTicks == 0 {
		return
	}
	
	// Calculate progress percentage (10% to 90% range, reserving 0-10% for initialization and 90-100% for finalization)
	progressPercent := 10.0 + (float64(pt.processedTicks)/float64(pt.totalTicks))*80.0
	if progressPercent > 90.0 {
		progressPercent = 90.0
	}
	
	message := fmt.Sprintf("Processing tick %d of %d", pt.processedTicks, pt.totalTicks)
	
	if err := pt.storage.UpdateRunProgress(pt.runID, progressPercent, message); err != nil {
		log.Printf("Failed to update progress: %v", err)
	}
	
	pt.lastProgressUpdate = time.Now()
}

func (pt *ProgressTrackingTrader) IncrementTickCount() {
	pt.progressMutex.Lock()
	pt.processedTicks++
	pt.progressMutex.Unlock()
	
	// Update progress more frequently during processing
	if time.Since(pt.lastProgressUpdate) > 5*time.Second {
		go pt.updateProgress()
	}
}