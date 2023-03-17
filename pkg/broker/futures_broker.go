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
	output       string
}

func NewFuturesBroker(c *config.BrokerConfig) IBroker {
	return &FuturesBroker{
		account: Account{
			balance:          c.StartingBalance,
			availableBalance: c.StartingBalance,
		},
		symbol: c.Symbol,
		output: "test.txt",
		trades: []float64{},
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
			b.account.availableBalance -= t.Price
			b.currentTrade = &t
			log.Printf("Updated Available Balance: %.2f\n", b.account.availableBalance)
		}
	case Sell:
		if b.currentTrade == nil {
			if !b.validTrade(&t) {
				return fmt.Errorf("Failed to create Sell: Insufficient account balance")
			}
			log.Printf("Broker Create Sell: Symbol %v Price: %.2f\n", t.Symbol, t.Price)
			b.account.availableBalance -= t.Price
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
		net = t.Price - b.currentTrade.Price
	case Sell:
		net = b.currentTrade.Price - t.Price
	}
	if net > 0 {
		b.wins++
		b.cumWin += net
	} else {
		b.loses++
		b.cumLose += net
	}
	b.account.balance += net
	b.account.availableBalance += (b.currentTrade.Price + net)
	b.trades = append(b.trades, net)
	fmt.Printf("Close trade at %v\n", time.Unix(t.Time, 0))
}

func (b *FuturesBroker) validTrade(t *Trade) bool {
	if b.account.availableBalance-200 < 0 {
		return false
	}
	return true
}

func (b *FuturesBroker) Summary() {
	log.Printf("Number of wins: %v\n", b.wins)
	log.Printf("Cumulative win: %v\n", b.cumWin)
	log.Printf("Number of loses: %v\n", b.loses)
	log.Printf("Cumulative lose: %v\n", b.cumLose)
	adjustedNet := b.cumWin*5 + b.cumLose*5
	adjustedNet -= float64((b.wins + b.loses) * 5)
	log.Printf("Adjusted net: $%.02f\n", adjustedNet)
	log.Printf("Trades taken: %v\n", b.trades)
}
