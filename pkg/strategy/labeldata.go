package strategy

import (
	"encoding/csv"
	"log"
	"os"
	"strconv"

	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/strategy/types"
)

type LabelStrategy struct {
	config        *types.Config
	indicatorChan chan types.Indicator
	data          datafeed.Data
	writer        *csv.Writer
	dataCache     []labeledData
}

type labeledData struct {
	timestamp int64
	price     float64
	upVar     float64
	downVar   float64
	label     string
}

func NewLabelStrategy(c *types.Config, ic chan types.Indicator) types.IStrategy {
	file, err := os.Create("data.csv")
	if err != nil {
		log.Fatal(err)
	}
	// create data cache

	// initialize csv writer
	writer := csv.NewWriter(file)
	header := []string{"timestamp", "close", "label"}
	writer.Write(header)
	writer.Flush()
	return &LabelStrategy{
		config:        c,
		indicatorChan: ic,
		writer:        writer,
	}
}

func (s *LabelStrategy) AddData(data datafeed.Data) error {
	s.data = data
	s.determineIndicator()
	return nil
}

func (s *LabelStrategy) GetIndicatorFeed() chan types.Indicator {
	return s.indicatorChan
}

func (s *LabelStrategy) determineIndicator() {
	// create strings from floats
	dateVal := strconv.FormatInt(s.data.Date, 10)
	closeVal := strconv.FormatFloat(s.data.Close, 'f', -1, 64)
	labelVal := "None"
	// write data
	data := []string{dateVal, closeVal, labelVal}
	s.writer.Write(data)
	s.writer.Flush()
	// send empty indicator
	s.indicatorChan <- types.Indicator{
		Direction: types.None,
	}
}
