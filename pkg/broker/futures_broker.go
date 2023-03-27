package broker

import (
	"fmt"
	"log"
	"time"

	"github.com/tradestax/traedor/internal/config"
	"github.com/tradestax/traedor/pkg/datafeed"
)

type FuturesBroker struct {
	account      *Account
	currentTrade *Trade
	config       *config.BrokerConfig
	currentPrice float64
	currentTime  int64
	stopAmount   float64
	feePerSide   float64
	wins         int
	loses        int
	cumWin       float64
	cumLose      float64
	trades       []*Trade
	margin       float64
	pointPrice   float64
	output       string
	week         int
}

func NewFuturesBroker(c *config.BrokerConfig) IBroker {
	return &FuturesBroker{
		account: &Account{
			balance:          c.StartingBalance,
			availableBalance: c.StartingBalance,
			weeklyWithdrawl:  c.WeeklyWithdrawl,
		},
		config:     c,
		output:     "trades.csv",
		trades:     make([]*Trade, 0),
		margin:     c.Symbol.Margin,
		pointPrice: c.Symbol.PointPrice,
		stopAmount: c.TrailingStopAmount,
		feePerSide: c.FeePerSide,
	}
}

func (b *FuturesBroker) GetAccountStats() (*Account, error) {
	return b.account, nil
}

func (b *FuturesBroker) AddData(d datafeed.Data) {
	b.currentPrice = d.Close
	b.currentTime = d.Date
	if b.currentTrade != nil {
		// update max values for trade
		b.updateTradeValues(d)
		if b.isStop() {
			b.closePosition()
		}
		// close all trades on Friday at 4:45pm
		now := time.Unix(d.Date, 0)
		if now.Weekday() == time.Friday {
			if now.Hour() >= 16 {
				if now.Minute() >= 45 {
					b.closePosition()
				}
			}
		}
	}
}

func (b *FuturesBroker) SendTrade(t Trade) error {
	tradeTime := time.Unix(t.Time, 0)
	if _, week := tradeTime.ISOWeek(); week != b.week {
		b.account.WeeklyWithdrawl()
		b.week = week
	}
	// close all trades on Friday at 4:45pm
	if tradeTime.Weekday() == time.Friday {
		if tradeTime.Hour() >= 16 {
			if tradeTime.Minute() >= 45 {
				log.Println("Broker not entering trade after 4:45pm on Friday")
				return nil
			}
		}
	}
	q := max(1, (b.account.availableBalance / ((b.margin) + b.feePerSide)))
	tradeQ := int(min(50, q))
	fee := float64(tradeQ) * b.feePerSide
	switch t.Operation {
	case Close:
		b.closePosition()
	case Buy:
		if b.currentTrade != nil && b.currentTrade.Operation == Sell {
			b.closePosition()
			return nil
		}
		if b.currentTrade == nil {
			if !b.validTrade(&t) {
				return fmt.Errorf("Failed to create Buy: Insufficient account balance")
			}
			log.Printf("Broker Create Buy: Symbol: %v Price: %.2f Quantity: %d\n", t.Symbol, b.currentPrice, tradeQ)
			b.account.availableBalance -= (float64(tradeQ) * b.margin)
			b.account.availableBalance -= fee
			b.account.balance -= fee
			b.currentTrade = &t
			b.currentTrade.Quantity = tradeQ
			// slippage
			b.currentTrade.Price = b.currentPrice + b.config.OpenSlippage
			b.currentTrade.ProfitPrice = b.currentTrade.Price + (b.stopAmount * 2)
			b.currentTrade.StopPrice = b.currentTrade.Price - b.stopAmount
			log.Printf("Updated Available Balance: %.2f\n", b.account.availableBalance)
		}
	case Sell:
		if b.currentTrade != nil && b.currentTrade.Operation == Buy {
			b.closePosition()
			return nil
		}
		if b.currentTrade == nil {
			if !b.validTrade(&t) {
				return fmt.Errorf("Failed to create Sell: Insufficient account balance")
			}
			log.Printf("Broker Create Sell: Symbol %v Price: %.2f Quantity: %d\n", t.Symbol, b.currentPrice, tradeQ)
			b.account.availableBalance -= (float64(tradeQ) * b.margin)
			b.account.availableBalance -= fee
			b.account.balance -= fee
			b.currentTrade = &t
			b.currentTrade.Quantity = tradeQ
			// slippage
			b.currentTrade.Price = b.currentPrice - b.config.OpenSlippage
			b.currentTrade.ProfitPrice = b.currentTrade.Price - (b.stopAmount * 2)
			b.currentTrade.StopPrice = b.currentTrade.Price + b.stopAmount
			log.Printf("Updated Available Balance: %.2f\n", b.account.availableBalance)
		}
	default:
		//fmt.Println("Broker No indicator")
		return nil
	}
	return nil
}

func (b *FuturesBroker) updateBalance() {
	var net float64
	multiplier := b.pointPrice * float64(b.currentTrade.Quantity)
	fee := float64(b.currentTrade.Quantity) * b.feePerSide
	switch b.currentTrade.Operation {
	case Buy:
		net = (b.currentPrice - b.currentTrade.Price) * multiplier
	case Sell:
		net = (b.currentTrade.Price - b.currentPrice) * multiplier
	}
	if net > 0 {
		b.wins++
		b.cumWin += net
	} else {
		b.loses++
		b.cumLose += net
	}
	// update current trade max values
	b.currentTrade.MaxProfit = b.currentTrade.MaxProfit * multiplier
	b.currentTrade.MaxDrawdown = b.currentTrade.MaxDrawdown * multiplier
	b.account.balance += net
	b.account.availableBalance += ((b.margin * float64(b.currentTrade.Quantity)) + net)
	b.account.availableBalance -= fee
	b.account.balance -= fee
	b.currentTrade.Net = net
	fmt.Printf("Close trade at %v\n", time.Unix(b.currentTime, 0))
}

func (b *FuturesBroker) closePosition() {
	if b.currentTrade == nil {
		log.Printf("Close called with no open position")
		return
	}
	log.Printf("Broker Close Trade: Symbol: %v Price: %.2f Quantity: %d\n", b.currentTrade.Symbol, b.currentPrice, b.currentTrade.Quantity)
	b.currentTrade.CloseTime = b.currentTime
	b.currentTrade.ClosePrice = b.currentPrice
	b.updateBalance()
	b.trades = append(b.trades, b.currentTrade)
	b.currentTrade = nil
	log.Printf("Updated Balance: %.2f Available Balance: %.2f\n", b.account.balance, b.account.availableBalance)
}

func (b *FuturesBroker) isStop() bool {
	if b.currentTrade.Operation == Buy {
		// check for take profit
		if b.currentPrice >= b.currentTrade.ProfitPrice {
			return true
		}
		// check for trailing stop cross
		if b.currentPrice <= b.currentTrade.StopPrice {
			return true
		}
		// update trailing stop
		b.currentTrade.StopPrice = max(b.currentTrade.StopPrice, b.currentPrice-b.stopAmount)
	} else if b.currentTrade.Operation == Sell {
		// check for take profit
		if b.currentPrice <= b.currentTrade.ProfitPrice {
			return true
		}
		// check for trailing stop cross
		if b.currentPrice >= b.currentTrade.StopPrice {
			return true
		}
		// update trailing stop
		b.currentTrade.StopPrice = min(b.currentTrade.StopPrice, b.currentPrice+b.stopAmount)
	} else {
		log.Printf("Somehow called close on no op trade")
	}
	return false
}

func (b *FuturesBroker) updateTradeValues(d datafeed.Data) {
	currDiff := d.Close - b.currentTrade.Price
	if b.currentTrade.Operation == Sell {
		currDiff = currDiff * -1
	}
	if currDiff > 0 {
		// trade is currently positive
		b.currentTrade.MaxProfit = max(b.currentTrade.MaxProfit, currDiff)
	} else {
		// trade is currently negative
		b.currentTrade.MaxDrawdown = min(b.currentTrade.MaxDrawdown, currDiff)
	}
}

func (b *FuturesBroker) validTrade(t *Trade) bool {
	if b.account.availableBalance-b.margin < 0 {
		return false
	}
	return true
}

func (b *FuturesBroker) Summary() {
	WriteTradesToFile(b.output, b.trades)
	log.Printf("Number of wins: %v\n", b.wins)
	log.Printf("Cumulative win amount: $%v\n", b.cumWin)
	log.Printf("Number of loses: %v\n", b.loses)
	log.Printf("Cumulative lose amount: $%v\n", b.cumLose)
	log.Printf("Number of trades: %v\n", len(b.trades))
	log.Printf("Net profit: $%v\n", (b.cumWin + b.cumLose))
	log.Printf("Net profit percentage: %.02f%%\n", ((b.cumWin+b.cumLose)/b.cumWin)*100)
	total := float64(b.wins + b.loses)
	log.Printf("Accuracy: %.02f%%\n", (float64(b.wins)/total)*100)
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
