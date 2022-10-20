package broker

import "fmt"

type Broker struct {
	account      Account
	currentTrade *Trade
	output       string
}

func NewBroker() *Broker {
	return &Broker{
		account: Account{
			balance:          100.0,
			availableBalance: 100.0,
		},
		output: "test.txt",
	}
}

func (b *Broker) GetAccountStats() (Account, error) {
	return b.account, nil
}

func (b *Broker) SendTrade(t Trade) error {
	switch t.Operation {
	case Close:
		if b.currentTrade != nil {
			fmt.Printf("Broker Close Trade: Symbol: %v Price: %.2f\n", t.Symbol, t.Price)
			b.updateBalance(t)
			b.currentTrade = nil
			fmt.Printf("Updated Balance: %.2f Available Balance: %.2f\n", b.account.balance, b.account.availableBalance)
		}
	case Buy:
		if b.currentTrade == nil {
			fmt.Printf("Broker Create Buy: Symbol: %v Price: %.2f\n", t.Symbol, t.Price)
			b.account.availableBalance -= t.Price
			b.currentTrade = &t
			fmt.Printf("Updated Available Balance: %.2f\n", b.account.availableBalance)
		}
	case Sell:
		if b.currentTrade == nil {
			fmt.Printf("Broker Create Sell: Symbol %v Price: %.2f\n", t.Symbol, t.Price)
			b.account.availableBalance -= t.Price
			b.currentTrade = &t
			fmt.Printf("Updated Available Balance: %.2f\n", b.account.availableBalance)
		}
	default:
		fmt.Println("Broker No indicator")
		return nil
	}
	return nil
}

func (b *Broker) updateBalance(t Trade) {
	var net float64
	switch b.currentTrade.Operation {
	case Buy:
		net = t.Price - b.currentTrade.Price
	case Sell:
		net = b.currentTrade.Price - t.Price
	}
	b.account.balance += net
	b.account.availableBalance += (b.currentTrade.Price + net)
}
