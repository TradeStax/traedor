package datafeed

import (
	"encoding/csv"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"
)

const (
	dateIndex  = 0
	openIndex  = 1
	highIndex  = 2
	lowIndex   = 3
	closeIndex = 4
	volIndex   = 5
)

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
	var rData Data
	for i, r := range records {
		rData, err = rowToData(r)
		if err != nil {
			if i == 0 {
				// its likely this is just the header row
				continue
			}
			panic(err)
		}
		if d.startTime == 0 {
			d.startTime = rData.Date
			startDate := time.Unix(rData.Date, 0)
			log.Printf("Start Time: %s\n", startDate)
		}
		d.dataChan <- rData
		time.Sleep(duration)
	}
	startDate := time.Unix(d.startTime, 0)
	log.Printf("Start Time: %s\n", startDate)
	endDate := time.Unix(rData.Date, 0)
	log.Printf("End Time: %s\n", endDate)
	d.errorChan <- fmt.Errorf("Test Completed")
}

func rowToData(r []string) (Data, error) {
	var d Data
	for i, v := range r {
		value, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return d, err
		}
		switch i {
		case dateIndex:
			d.Date = int64(value)/1000
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
