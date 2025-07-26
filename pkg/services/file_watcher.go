package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tradestax/traedor/pkg/storage"
)

type FileWatcher struct {
	dataDir    string
	storage    storage.IStorage
	importPool *ImportPool
	interval   time.Duration
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewFileWatcher(dataDir string, store storage.IStorage) *FileWatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &FileWatcher{
		dataDir:    dataDir,
		storage:    store,
		importPool: NewImportPool(store),
		interval:   30 * time.Second, // Check every 30 seconds
		ctx:        ctx,
		cancel:     cancel,
	}
}

func (fw *FileWatcher) Start() {
	log.Printf("Starting file watcher for directory: %s", fw.dataDir)
	
	// Check for and resume stuck imports from previous runs
	fw.resumeStuckImports()
	
	// Initial scan
	fw.scanAndImport()
	
	// Start periodic scanning
	ticker := time.NewTicker(fw.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				fw.scanAndImport()
			case <-fw.ctx.Done():
				log.Println("File watcher stopped")
				return
			}
		}
	}()
}

func (fw *FileWatcher) Stop() {
	log.Println("Stopping file watcher...")
	if fw.cancel != nil {
		fw.cancel()
	}
	fw.importPool.Stop() // Wait for all imports to complete
}

func (fw *FileWatcher) scanAndImport() {
	log.Printf("Scanning directory: %s", fw.dataDir)
	
	// Get all files once at the beginning to reduce DB queries
	existingFiles, err := fw.storage.ListMarketDataFiles()
	if err != nil {
		log.Printf("Error listing existing files: %v", err)
		return
	}
	
	// Create hash lookup map for faster checking
	hashLookup := make(map[string]*storage.MarketDataFile)
	for _, f := range existingFiles {
		hashLookup[f.FileHash] = f
	}
	
	// Walk through the data directory
	err = filepath.Walk(fw.dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories and non-data files
		if info.IsDir() {
			return nil
		}
		
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if ext != ".txt" && ext != ".csv" {
			return nil
		}
		
		// Skip small files (likely config files)
		if info.Size() < 1024 { // Less than 1KB
			return nil
		}
		
		// Calculate file hash to check if it's already processed
		hash, err := fw.calculateFileHash(path)
		if err != nil {
			log.Printf("Error calculating hash for %s: %v", path, err)
			return nil
		}
		
		// Check if file is already imported or being processed using cached lookup
		if existingFile, exists := hashLookup[hash]; exists {
			log.Printf("File %s found in database with status: %s", info.Name(), existingFile.Status)
			if existingFile.Status == "completed" {
				return nil
			} else if existingFile.Status == "processing" {
				return nil
			} else if existingFile.Status == "pending" {
				return nil
			} else if existingFile.Status == "failed" {
				log.Printf("File %s previously failed import, skipping automatic retry", info.Name())
				return nil
			}
			// If we get here, it's an unknown status - log it
			log.Printf("WARNING: File %s has unknown status: %s", info.Name(), existingFile.Status)
		}
		
		// Start import using pool to prevent goroutine leaks
		log.Printf("Starting import of new file: %s (%.2f MB)", info.Name(), float64(info.Size())/1024/1024)
		fw.importPool.ImportFileAsync(path)
		
		return nil
	})
	
	if err != nil {
		log.Printf("Error scanning directory: %v", err)
	}
}

func (fw *FileWatcher) calculateFileHash(filePath string) (string, error) {
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

// ManualImport allows manual triggering of a specific file import
func (fw *FileWatcher) ManualImport(filePath string) error {
	log.Printf("Manual import requested for: %s", filePath)
	fw.importPool.ImportFileAsync(filePath)
	return nil
}

// resumeStuckImports checks for imports that were stuck in processing/pending state
// and either resumes them or resets them based on file existence
func (fw *FileWatcher) resumeStuckImports() {
	log.Println("Checking for stuck imports from previous runs...")
	
	stuckImports, err := fw.storage.GetStuckImports()
	if err != nil {
		log.Printf("Error getting stuck imports: %v", err)
		return
	}
	
	if len(stuckImports) == 0 {
		log.Println("No stuck imports found")
		return
	}
	
	log.Printf("Found %d stuck imports, checking file availability...", len(stuckImports))
	
	// Deduplicate by file hash to prevent duplicate processing
	// Keep only the most recent record for each unique file
	uniqueImports := make(map[string]*storage.MarketDataFile)
	duplicateIDs := make([]string, 0)
	
	for _, file := range stuckImports {
		if existing, exists := uniqueImports[file.FileHash]; exists {
			// We have a duplicate - keep the more recent one and mark the older as failed
			if file.CreatedAt.After(existing.CreatedAt) {
				// Current file is newer, mark the existing one as failed and replace it
				duplicateIDs = append(duplicateIDs, existing.ID)
				uniqueImports[file.FileHash] = file
				log.Printf("Found duplicate stuck import for file %s, keeping newer record (ID: %s), marking older as failed (ID: %s)", 
					file.Filename, file.ID, existing.ID)
			} else {
				// Existing file is newer, mark current one as failed
				duplicateIDs = append(duplicateIDs, file.ID)
				log.Printf("Found duplicate stuck import for file %s, keeping existing record (ID: %s), marking older as failed (ID: %s)", 
					file.Filename, existing.ID, file.ID)
			}
		} else {
			uniqueImports[file.FileHash] = file
		}
	}
	
	// Mark duplicate records as failed to prevent future conflicts
	for _, duplicateID := range duplicateIDs {
		if err := fw.storage.UpdateMarketDataFileStatus(duplicateID, "failed", "Duplicate import record removed during deduplication"); err != nil {
			log.Printf("Error marking duplicate import as failed (ID: %s): %v", duplicateID, err)
		}
	}
	
	if len(duplicateIDs) > 0 {
		log.Printf("Marked %d duplicate import records as failed", len(duplicateIDs))
	}
	
	var resumedCount, resetCount int
	
	// Process only the unique imports
	for _, file := range uniqueImports {
		// Check if the file still exists
		if _, err := os.Stat(file.FilePath); err == nil {
			// File exists, resume the import
			log.Printf("Resuming import for: %s (ID: %s)", file.Filename, file.ID)
			fw.importPool.ImportFileAsync(file.FilePath)
			resumedCount++
		} else {
			// File doesn't exist, reset to failed
			log.Printf("File %s no longer exists, marking as failed (ID: %s)", file.Filename, file.ID)
			if err := fw.storage.UpdateMarketDataFileStatus(file.ID, "failed", "File no longer exists after server restart"); err != nil {
				log.Printf("Error updating file status: %v", err)
			} else {
				resetCount++
			}
		}
	}
	
	log.Printf("Import recovery complete: %d resumed, %d reset to failed, %d duplicates removed", resumedCount, resetCount, len(duplicateIDs))
}

// CleanupStuckImports manually resets all stuck imports - useful for maintenance
func (fw *FileWatcher) CleanupStuckImports() (int, error) {
	log.Println("Manually cleaning up all stuck imports...")
	count, err := fw.storage.ResetStuckImports()
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup stuck imports: %w", err)
	}
	log.Printf("Reset %d stuck imports", count)
	return count, nil
}

