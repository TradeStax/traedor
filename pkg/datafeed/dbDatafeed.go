package datafeed

import (
	"context"
	"fmt"
	"time"

	"github.com/tradestax/traedor/pkg/storage"
)

type DBDatafeed struct {
	config    *Config
	storage   storage.IStorage
	dataChan  chan Data
	errorChan chan error
	ctx       context.Context
	cancel    context.CancelFunc
}

func NewDBDatafeed(config *Config, store storage.IStorage, dataChan chan Data, errorChan chan error) *DBDatafeed {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &DBDatafeed{
		config:    config,
		storage:   store,
		dataChan:  dataChan,
		errorChan: errorChan,
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (d *DBDatafeed) GetDatafeed() chan Data {
	return d.dataChan
}

func (d *DBDatafeed) GetErrorChan() chan error {
	return d.errorChan
}

func (d *DBDatafeed) Start() {
	fmt.Printf("DBDatafeed: Starting database datafeed for symbol %s from %v to %v\n", d.config.Symbol, time.Unix(d.config.StartTime, 0), time.Unix(d.config.EndTime, 0))
	go d.stream()
}

func (d *DBDatafeed) stream() {
	// Convert timestamps
	startTime := time.Unix(d.config.StartTime, 0)
	endTime := time.Unix(d.config.EndTime, 0)
	
	fmt.Printf("DBDatafeed: Starting true streaming for %s from %v to %v\n", d.config.Symbol, startTime, endTime)
	
	totalProcessed := 0
	
	// Use the new streaming method that processes one tick at a time
	err := d.storage.StreamOHLCData(d.config.Symbol, startTime, endTime, func(ohlc storage.OHLCData) error {
		totalProcessed++
		
		// Log progress every 10,000 ticks
		if totalProcessed%10000 == 0 {
			fmt.Printf("DBDatafeed: Streamed %d ticks...\n", totalProcessed)
		}
		
		select {
		case <-d.ctx.Done():
			return fmt.Errorf("context cancelled")
		default:
			// Convert prices from database storage format (multiplied by 100) to actual prices
			data := Data{
				Symbol: ohlc.Symbol,
				Date:   ohlc.Time.Unix(), // Unix seconds - aggregator expects this
				Open:   ohlc.Open / 100.0,
				High:   ohlc.High / 100.0,
				Low:    ohlc.Low / 100.0,
				Close:  ohlc.Close / 100.0,
				Volume: float64(ohlc.Volume),
			}
			
			// Send tick data directly to the channel
			select {
			case d.dataChan <- data:
			case <-d.ctx.Done():
				return fmt.Errorf("context cancelled")
			}
		}
		
		return nil
	})
	
	if err != nil {
		d.errorChan <- fmt.Errorf("error streaming OHLC data: %w", err)
		return
	}
	
	fmt.Printf("DBDatafeed: Completed streaming %d total records for symbol %s\n", totalProcessed, d.config.Symbol)
	
	// Signal completion by closing the data channel
	close(d.dataChan)
}

func (d *DBDatafeed) Stop() error {
	d.cancel()
	return nil
}