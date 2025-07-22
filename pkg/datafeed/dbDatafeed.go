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
	go d.stream()
}

func (d *DBDatafeed) stream() {
	// Convert timestamps
	startTime := time.Unix(d.config.StartTime, 0)
	endTime := time.Unix(d.config.EndTime, 0)
	
	// Fetch OHLC data from database
	ohlcData, err := d.storage.GetOHLCData(d.config.Symbol, startTime, endTime)
	if err != nil {
		d.errorChan <- fmt.Errorf("error fetching OHLC data: %w", err)
		return
	}
	
	// Convert OHLC data to datafeed format and stream
	for _, ohlc := range ohlcData {
		select {
		case <-d.ctx.Done():
			return
		default:
			// For tick data (tick_sequence > 0), use close price as the tick price
			// For bar data, we could emit OHLC values or just close
			data := Data{
				Symbol: ohlc.Symbol,
				Date:   ohlc.Time.Unix(),
				Open:   ohlc.Open,
				High:   ohlc.High,
				Low:    ohlc.Low,
				Close:  ohlc.Close,
				Volume: float64(ohlc.Volume),
			}
			
			select {
			case d.dataChan <- data:
			case <-d.ctx.Done():
				return
			}
		}
	}
	
	// Signal completion by closing the data channel
	close(d.dataChan)
}

func (d *DBDatafeed) Stop() error {
	d.cancel()
	return nil
}