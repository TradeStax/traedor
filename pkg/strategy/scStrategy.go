package strategy

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/strategy/types"
)

const (
	scDateLayout = "2006-1-2 15:04:05.000000"
)

type ScStrategy struct {
	config        *types.Config
	dataCache     []datafeed.Data
	indicatorChan chan types.Indicator
	headers       map[string]int
	values        types.Values
	r             *csv.Reader
	lastSend      types.Indicator
	stop          float64
	currStop      float64
}

func NewScStrategy(c *types.Config, ic chan types.Indicator) types.IStrategy {
	f, err := os.Open(c.Params.DataPath)
	if err != nil {
		panic(fmt.Errorf("Failed to open datapath"))
	}
	csvReader := csv.NewReader(f)
	headerRow, err := csvReader.Read()
	if err != nil {
		panic(err)
	}
	headers := make(map[string]int, len(headerRow))
	for i, v := range headerRow {
		v = strings.Trim(v, " ")
		headers[v] = i
	}

	for _, v := range c.Params.Values {
		if _, ok := headers[v]; !ok {
			log.Fatalf("Failed to find value %v in data header row\n", v)
		}
	}
	return &ScStrategy{
		config:        c,
		dataCache:     make([]datafeed.Data, 10),
		indicatorChan: ic,
		headers:       headers,
		values: types.Values{
			Studies: map[string]float64{},
		},
		r:    csvReader,
		stop: float64(10.0),
	}
}

func (s *ScStrategy) AddData(data datafeed.Data) error {
	if data.Close == s.dataCache[9].Close {
		s.indicatorChan <- types.Indicator{Direction: types.None}
		return nil
	}
	for i := 0; i < len(s.dataCache)-1; i++ {
		s.dataCache[i] = s.dataCache[i+1]
	}
	s.dataCache[9] = data
	s.determineIndicator()
	return nil
}

func (s *ScStrategy) GetIndicatorFeed() chan types.Indicator {
	return s.indicatorChan
}

func (s *ScStrategy) determineIndicator() {
	ind := types.Indicator{
		Price:     s.dataCache[9].Close,
		Time:      s.dataCache[9].Date,
		Direction: types.None,
	}
	s.indicatorChan <- ind
}
