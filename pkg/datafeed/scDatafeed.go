package datafeed

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	scDateLayout     = "2006/1/2 15:04:05.000"
	scDateLayoutNoMs = "2006/1/2 15:04:05"
)

func (d *Datafeed) scDatafeed(duration time.Duration) {
	f, err := os.Open(d.config.DataPath)
	if err != nil {
		panic(fmt.Errorf("Failed to open datapath"))
	}
	defer f.Close()
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
	var rData Data
	i := 0
	for {
		record, err := csvReader.Read()
		// Stop at EOF.
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}
		rData, err = scRowToData(headers, record)
		if err != nil {
			if i == 0 {
				i++
				// its likely this is just the header row
				continue
			}
			panic(err)
		}
		if d.startTime == 0 {
			d.startTime = rData.Date
			startDate := time.Unix(rData.Date, 0)
			log.Printf("Start Time: %s\n", startDate)
		} else if rData.Date < d.startTime {
			// skip records before start time
			continue
		} else if d.endTime > 0 && rData.Date > d.endTime {
			// skip records past end time
			break
		}
		if d.config.CompressRecords && d.prev == rData.Close {
			d.skipped++
			continue
		}
		d.prev = rData.Close
		d.dataChan <- rData
		time.Sleep(duration)
	}
	startDate := time.Unix(d.startTime, 0)
	log.Printf("Start Time: %s\n", startDate)
	endDate := time.Unix(rData.Date, 0)
	log.Printf("End Time: %s\n", endDate)
	log.Printf("Skipped %d records\n", d.skipped)
	d.errorChan <- fmt.Errorf("Test Completed")
}

func scRowToData(headers map[string]int, r []string) (Data, error) {
	var d Data
	t, err := time.Parse(scDateLayout, r[0]+r[1])
	//time, err := time.Parse(scDateLayout, r[0])
	if err != nil {
		t, err = time.Parse(scDateLayoutNoMs, r[0]+r[1])
		if err != nil {
			return d, err
		}
	}
	d.Date = t.Unix()
	closeValue, err := strconv.ParseFloat(strings.Trim(r[headers["Last"]], " "), 64)
	if err != nil {
		return d, err
	}
	d.Close = closeValue / 100
	volValue, err := strconv.ParseFloat(strings.Trim(r[headers["NumberOfTrades"]], " "), 64)
	if err != nil {
		return d, err
	}
	d.Volume = volValue
	/*
		for i, v := range r {
			if i < 2 {
				continue
			}
			v := strings.Trim(v, " ")
			value, err := strconv.ParseFloat(v, 64)
			value = value / 100
			if err != nil {
				return d, err
			}
			switch i {
			case 3:
				d.Close = value
			case 6:
				d.Volume = value
				//default:
				//return d, fmt.Errorf("Unknown index")
			}
		}
	*/
	return d, nil
}
