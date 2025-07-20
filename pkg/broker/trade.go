package broker

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/tradestax/traedor/pkg/broker/profit"
	"github.com/tradestax/traedor/pkg/broker/stop"
)

const (
	None  = 0
	Close = 1
	Buy   = 2
	Sell  = 3
)

type Trade struct {
	Symbol      string
	Operation   int
	Quantity    int
	Price       float64
	OpenPrice   float64  // Alias for Price for clarity
	ClosePrice  float64
	Stops       []stop.IStop
	Profits     []profit.IProfit
	Time        int64
	OpenTime    int64     // Alias for Time for clarity
	CloseTime   int64
	Net         float64
	NetProfit   float64  // Alias for Net for clarity
	MaxDrawdown float64
	MaxProfit   float64
	MFE         float64  // Maximum Favorable Excursion
	MAE         float64  // Maximum Adverse Excursion
	MFEPercent  float64 // MFE as percentage
	MAEPercent  float64 // MAE as percentage
}

func (t *Trade) String() string {
	values := make([]string, 14)
	values[0] = t.Symbol
	if t.Operation == Buy {
		values[1] = "Buy"
	} else if t.Operation == Sell {
		values[1] = "Sell"
	} else {
		values[1] = "N/A"
	}
	values[2] = fmt.Sprintf("%d", t.Quantity)
	values[3] = fmt.Sprintf("%.02f", t.Price)
	values[4] = fmt.Sprintf("%.02f", t.ClosePrice)
	values[5] = fmt.Sprintf("%d", t.Time)
	values[6] = fmt.Sprintf("%d", t.CloseTime)
	values[7] = fmt.Sprintf("%.02f", t.Net)
	values[8] = fmt.Sprintf("%.02f", t.MaxDrawdown)
	values[9] = fmt.Sprintf("%.02f", t.MaxProfit)
	values[10] = fmt.Sprintf("%.02f", t.MFE)
	values[11] = fmt.Sprintf("%.02f", t.MFEPercent)
	values[12] = fmt.Sprintf("%.02f", t.MAE)
	values[13] = fmt.Sprintf("%.02f", t.MAEPercent)
	return strings.Join(values, ",")
}

func Header() string {
	values := make([]string, 14)
	values[0] = "Symbol"
	values[1] = "Operation"
	values[2] = "Quantity"
	values[3] = "Open Price"
	values[4] = "Close Price"
	values[5] = "Open Time"
	values[6] = "Close Time"
	values[7] = "Net"
	values[8] = "Max Drawdown"
	values[9] = "Max Profit"
	values[10] = "MFE"
	values[11] = "MFE %"
	values[12] = "MAE"
	values[13] = "MAE %"
	return strings.Join(values, ",")
}

func WriteTradesToFile(filename string, trades []*Trade) {
	f, err := os.Create(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	_, err2 := f.WriteString(Header() + "\n")
	if err2 != nil {
		log.Fatal(err2)
	}
	for _, t := range trades {
		_, err3 := f.WriteString(t.String() + "\n")
		if err3 != nil {
			log.Fatal(err3)
		}
	}
}
