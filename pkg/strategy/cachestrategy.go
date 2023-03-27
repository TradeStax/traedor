package strategy

import (
	"encoding/csv"
	"os"
	"log"
	"strconv"
	"github.com/tradestax/traedor/internal/config"
	"github.com/tradestax/traedor/pkg/datafeed"
)

type CacheStrategy struct {
	config        config.StrategyConfig
	indicatorChan chan Indicator
	data datafeed.Data
	writer *csv.Writer
}

func NewCacheStrategy(c config.StrategyConfig, ic chan Indicator) IStrategy {
	file, err := os.Create("data.csv")
    if err != nil {
        log.Fatal(err)
    }
    // initialize csv writer
    writer := csv.NewWriter(file)
	return &CacheStrategy{
		config:        c,
		indicatorChan: ic,
		writer: writer,
	}
}

func (s *CacheStrategy) AddData(data datafeed.Data) error {
	s.data = data
	s.determineIndicator()
	return nil
}

func (s *CacheStrategy) GetIndicatorFeed() chan Indicator {
	return s.indicatorChan
}

func (s *CacheStrategy) determineIndicator() {
	// create strings from floats
	dateVal := strconv.FormatInt(s.data.Date, 10)
	openVal := strconv.FormatFloat(s.data.Open, 'f', -1, 64)
	highVal := strconv.FormatFloat(s.data.High, 'f', -1, 64)
	lowVal := strconv.FormatFloat(s.data.Low, 'f', -1, 64)
	closeVal := strconv.FormatFloat(s.data.Close, 'f', -1, 64)
	volVal := strconv.FormatFloat(s.data.Volume, 'f', -1, 64)
	// write data
	data := []string{dateVal, openVal,highVal,lowVal,closeVal,volVal}
	s.writer.Write(data)
	s.writer.Flush()
	// send empty indicator
	s.indicatorChan <- Indicator{
		Direction: None,
	}
}
