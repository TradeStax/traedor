package broker

import (
	"fmt"
	"log"
	"time"

	"github.com/tradestax/traedor/internal/config"
)

type FuturesBroker struct {
	account      Account
	currentTrade *Trade
	wins         int
	loses        int
	cumWin       float64
	cumLose      float64
	trades       []float64
	symbol       string
	margin       float64
	pointPrice   float64
	output       string
}

func NewFuturesBroker(c *config.BrokerConfig) IBroker {
	return &FuturesBroker{
		account: Account{
			balance:          c.StartingBalance,
			availableBalance: c.StartingBalance,
		},
		symbol:     c.Symbol.Name,
		output:     "test.txt",
		trades:     []float64{},
		margin:     c.Symbol.Margin,
		pointPrice: c.Symbol.PointPrice,
	}
}

func (b *FuturesBroker) GetAccountStats() (Account, error) {
	return b.account, nil
}

func (b *FuturesBroker) SendTrade(t Trade) error {
	switch t.Operation {
	case Close:
		if b.currentTrade != nil {
			log.Printf("Broker Close Trade: Symbol: %v Price: %.2f\n", t.Symbol, t.Price)
			b.updateBalance(t)
			b.currentTrade = nil
			log.Printf("Updated Balance: %.2f Available Balance: %.2f\n", b.account.balance, b.account.availableBalance)
		}
	case Buy:
		if b.currentTrade == nil {
			if !b.validTrade(&t) {
				return fmt.Errorf("Failed to create Buy: Insufficient account balance")
			}
			log.Printf("Broker Create Buy: Symbol: %v Price: %.2f\n", t.Symbol, t.Price)
			b.account.availableBalance -= b.margin
			b.currentTrade = &t
			log.Printf("Updated Available Balance: %.2f\n", b.account.availableBalance)
		}
	case Sell:
		if b.currentTrade == nil {
			if !b.validTrade(&t) {
				return fmt.Errorf("Failed to create Sell: Insufficient account balance")
			}
			log.Printf("Broker Create Sell: Symbol %v Price: %.2f\n", t.Symbol, t.Price)
			b.account.availableBalance -= b.margin
			b.currentTrade = &t
			log.Printf("Updated Available Balance: %.2f\n", b.account.availableBalance)
		}
	default:
		//fmt.Println("Broker No indicator")
		return nil
	}
	return nil
}

func (b *FuturesBroker) updateBalance(t Trade) {
	var net float64
	switch b.currentTrade.Operation {
	case Buy:
		net = (t.Price - b.currentTrade.Price) * b.pointPrice
	case Sell:
		net = (b.currentTrade.Price - t.Price) * b.pointPrice
	}
	if net > 0 {
		b.wins++
		b.cumWin += net
	} else {
		b.loses++
		b.cumLose += net
	}
	b.account.balance += net
	b.account.availableBalance += (b.margin + net)
	b.trades = append(b.trades, net)
	fmt.Printf("Close trade at %v\n", time.Unix(t.Time, 0))
}

func (b *FuturesBroker) validTrade(t *Trade) bool {
	if b.account.availableBalance-b.margin < 0 {
		return false
	}
	return true
}

func (b *FuturesBroker) Summary() {
	log.Printf("Number of wins: %v\n", b.wins)
	log.Printf("Cumulative win amount: $%v\n", b.cumWin)
	log.Printf("Number of loses: %v\n", b.loses)
	log.Printf("Cumulative lose amount: $%v\n", b.cumLose)
	// net := float64(0.0)
	// for _, t := range b.trades {
	//	net += t
	// }
	log.Printf("Number of trades: %v\n", len(b.trades))
	log.Printf("Net profit: $%v\n", (b.cumWin + b.cumLose))
	total := float64(b.wins + b.loses)
	log.Printf("Accuracy: %.02f%%\n", (float64(b.wins)/total)*100)
}
