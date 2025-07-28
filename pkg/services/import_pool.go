package services

import (
	"context"
	"log"
	"sync"

	"github.com/tradestax/traedor/pkg/importer"
	"github.com/tradestax/traedor/pkg/storage"
)

const maxImportWorkers = 3 // Limit concurrent imports

type ImportPool struct {
	storage   storage.IStorage
	semaphore chan struct{}
	wg        sync.WaitGroup
}

func NewImportPool(storage storage.IStorage) *ImportPool {
	return &ImportPool{
		storage:   storage,
		semaphore: make(chan struct{}, maxImportWorkers),
	}
}

func (ip *ImportPool) ImportFileAsync(filePath string) {
	ip.wg.Add(1)
	go func() {
		defer ip.wg.Done()
		
		// Acquire semaphore (limit concurrent imports)
		ip.semaphore <- struct{}{}
		defer func() { <-ip.semaphore }()
		
		imp := importer.NewImporter(ip.storage)
		ctx := context.Background()
		if err := imp.ImportFile(ctx, filePath); err != nil {
			log.Printf("Import failed for %s: %v", filePath, err)
		} else {
			log.Printf("Import completed for %s", filePath)
		}
	}()
}

func (ip *ImportPool) Stop() {
	ip.wg.Wait()
}