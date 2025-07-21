package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tradestax/traedor/pkg/importer"
	"github.com/tradestax/traedor/pkg/storage"
)

type FileWatcher struct {
	dataDir   string
	storage   storage.IStorage
	importer  *importer.Importer
	interval  time.Duration
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewFileWatcher(dataDir string, store storage.IStorage) *FileWatcher {
	ctx, cancel := context.WithCancel(context.Background())
	return &FileWatcher{
		dataDir:  dataDir,
		storage:  store,
		importer: importer.NewImporter(store),
		interval: 30 * time.Second, // Check every 30 seconds
		ctx:      ctx,
		cancel:   cancel,
	}
}

func (fw *FileWatcher) Start() {
	log.Printf("Starting file watcher for directory: %s", fw.dataDir)
	
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
}

func (fw *FileWatcher) scanAndImport() {
	log.Printf("Scanning directory: %s", fw.dataDir)
	
	// Walk through the data directory
	err := filepath.Walk(fw.dataDir, func(path string, info os.FileInfo, err error) error {
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
		
		// Check if file is already imported
		exists, err := fw.storage.FileAlreadyImported(hash)
		if err != nil {
			log.Printf("Error checking if file exists: %v", err)
			return nil
		}
		
		if exists {
			log.Printf("File %s already imported, skipping", info.Name())
			return nil
		}
		
		// Check if file is already being processed
		files, err := fw.storage.ListMarketDataFiles()
		if err != nil {
			log.Printf("Error listing files: %v", err)
			return nil
		}
		
		for _, f := range files {
			if f.FileHash == hash {
				if f.Status == "processing" {
					log.Printf("File %s is already being processed", info.Name())
					return nil
				} else if f.Status == "failed" {
					log.Printf("File %s previously failed import, skipping automatic retry", info.Name())
					return nil
				}
			}
		}
		
		// Start import
		log.Printf("Starting import of new file: %s (%.2f MB)", info.Name(), float64(info.Size())/1024/1024)
		go func(filePath string) {
			if err := fw.importer.ImportFile(fw.ctx, filePath); err != nil {
				log.Printf("Import failed for %s: %v", filePath, err)
			} else {
				log.Printf("Import completed for %s", filepath.Base(filePath))
			}
		}(path)
		
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
	return fw.importer.ImportFile(fw.ctx, filePath)
}