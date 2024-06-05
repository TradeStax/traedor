package strategy

import (
	"encoding/csv"
	"log"
	"os"
	"strconv"

	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/strategy/types"
)

const (
	profit = 10
	loss   = -5
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
	long      *trade
	short     *trade
	label     string
}

type trade struct {
	open      float64
	loss      float64
	profit    float64
	direction int
	/*
		from types package
		None  = 0
		Close = 1
		Buy   = 2
		Sell  = 3
	*/
}

func (t *trade) update(p float64) {
	if t.direction == types.Buy {
		t.loss = min(t.loss, (p - t.open))
		t.profit = max(t.profit, (p - t.open))
	} else {
		t.loss = min(t.loss, (t.open - p))
		t.profit = max(t.profit, (t.open - p))
	}
}

func (t *trade) complete() bool {
	return t.profit >= profit || t.loss <= loss
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
	s.dataCache = append(s.dataCache, labeledData{
		timestamp: data.Date,
		price:     data.Close,
		long: &trade{
			open:      data.Close,
			direction: types.Buy,
		},
		short: &trade{
			open:      data.Close,
			direction: types.Sell,
		},
		label: "None",
	})
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
