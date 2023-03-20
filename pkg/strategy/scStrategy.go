package strategy

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/tradestax/traedor/internal/config"
	"github.com/tradestax/traedor/pkg/datafeed"
)

const (
	scDateLayout = "2006-1-2 15:04:05.000000"
)

type ScStrategy struct {
	config         config.StrategyConfig
	dataCache      []datafeed.Data
	indicatorChan  chan Indicator
	headers        map[string]int
	values         Values
	candleDuration int64
	r              *csv.Reader
	lastSend       Indicator
	stop           float64
	currStop       float64
}

type Values struct {
	studies   map[string]float64
	timestamp int64
}

type cross struct {
	above bool
	below bool
}

func NewScStrategy(c config.StrategyConfig, ic chan Indicator) IStrategy {
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
		values: Values{
			studies: map[string]float64{},
		},
		candleDuration: int64(c.Params.Timeframe.Seconds()),
		r:              csvReader,
		stop:           float64(2.0),
	}
}

func (s *ScStrategy) AddData(data datafeed.Data) error {
	for i := 0; i < len(s.dataCache)-1; i++ {
		s.dataCache[i] = s.dataCache[i+1]
	}
	s.dataCache[9] = data
	s.getStudies(data.Date)
	s.determineIndicator()
	return nil
}

func (s *ScStrategy) GetIndicatorFeed() chan Indicator {
	return s.indicatorChan
}

func (s *ScStrategy) determineIndicator() {
	ind := Indicator{
		Price: s.dataCache[9].Close,
		Time:  s.dataCache[9].Date,
	}
	crossDir := isCross(s.values.studies["12B"], s.dataCache)
	if crossDir.below {
		ind.Direction = Sell
		s.lastSend = ind
	} else if crossDir.above {
		ind.Direction = Buy
		s.lastSend = ind
	} else if s.isStop() {
		ind.Direction = Close
		s.lastSend = ind
	}
	s.indicatorChan <- ind
}

func (s *ScStrategy) isStop() bool {
	currPrice := s.dataCache[9].Close
	if s.lastSend.Direction == Sell {
		if currPrice >= s.currStop {
			return true
		}
		s.currStop = minf(s.currStop, currPrice+s.stop)
	} else if s.lastSend.Direction == Buy {
		if currPrice <= s.currStop {
			return true
		}
		s.currStop = maxf(s.currStop, currPrice-s.stop)
	}
	return false
}

func (s *ScStrategy) getStudies(date int64) {
	var newValues Values
	for {
		// values are already current
		if date < s.values.timestamp+s.candleDuration {
			break
		}
		// need to get new values
		values, err := s.r.Read()
		if err != nil {
			if err == io.EOF {
				// log.Println("No more values to read, using last values")
				break
			}
			panic(err)
		}
		t, err := time.Parse(scDateLayout, values[0]+values[1])
		if err != nil {
			log.Fatalf("%v\n", err.Error())
		}
		studies := make(map[string]float64, len(s.config.Params.Values))
		for _, v := range s.config.Params.Values {
			value := strings.Trim(values[s.headers[v]], " ")
			sv, err := strconv.ParseFloat(value, 64)
			if err != nil {
				log.Fatalf("%v\n", err.Error())
			}
			studies[v] = sv
		}
		newValues = Values{
			timestamp: t.Unix(),
			studies:   studies,
		}
		s.values = newValues
	}
}

func isCross(point float64, data []datafeed.Data) cross {
	var dir cross
	start := point - data[0].Close
	for _, d := range data {
		diff := point - d.Close
		if start < 0 && start-diff < start {
			dir.above = true
			return dir
		} else if start > 0 && start-diff > start {
			dir.below = true
			return dir
		}
	}
	return dir
}

func maxf(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minf(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
