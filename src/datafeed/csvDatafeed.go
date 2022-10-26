package datafeed

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/tradestax/traedor/config"
)

const (
	dateIndex  = 0
	openIndex  = 1
	highIndex  = 2
	lowIndex   = 3
	closeIndex = 4
	volIndex   = 5
)

func NewCSVDatafeed(c *config.Config) *Datafeed {
	duration, err := time.ParseDuration(c.Interval)
	if err != nil {
		panic(err)
	}
	df := &Datafeed{
		config:    c,
		dataChan:  make(chan Data),
		errorChan: make(chan error),
	}
	go df.csvDatafeed(duration)
	return df
}

func (d *Datafeed) csvDatafeed(duration time.Duration) {
	f, err := os.Open(d.config.DataPath)
	if err != nil {
		panic(fmt.Errorf("Failed to open datapath"))
	}
	defer f.Close()
	csvReader := csv.NewReader(f)
	records, err := csvReader.ReadAll()
	if err != nil {
		panic(err)
	}
	for i, r := range records {
		if i == 0 {
			continue
		}
		rData, err := rowToData(r)
		if err != nil {
			panic(err)
		}
		d.dataChan <- rData
		time.Sleep(duration)
	}
	d.errorChan <- fmt.Errorf("Test Completed")
}

func rowToData(r []string) (Data, error) {
	var d Data
	for i, v := range r {
		if i == dateIndex {
			continue
		}
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return d, err
		}
		switch i {
		case highIndex:
			d.High = value
		case lowIndex:
			d.Low = value
		case openIndex:
			d.Open = value
		case closeIndex:
			d.Close = value
		case volIndex:
			d.Volume = value
		default:
			return d, fmt.Errorf("Unknown index")
		}
	}
	return d, nil
}
