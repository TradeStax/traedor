package broker

import (
	"fmt"
	"log"
	"os"
	"strings"
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
	ClosePrice  float64
	StopPrice   float64
	ProfitPrice float64
	Time        int64
	CloseTime   int64
	Net         float64
	MaxDrawdown float64
	MaxProfit   float64
}

func (t *Trade) String() string {
	values := make([]string, 10)
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
	return strings.Join(values, ",")
}

func Header() string {
	values := make([]string, 10)
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
