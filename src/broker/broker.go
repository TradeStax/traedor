package broker

import "fmt"

type Broker struct {
	output string
}

func NewBroker() *Broker {
	return &Broker{
		output: "test.txt",
	}
}

func (b *Broker) GetAccountStats() (Account, error) {
	return Account{}, nil
}

func (b *Broker) SendTrade(t Trade) error {
	switch t.Operation {
	case Close:
		fmt.Println("Broker Close Trade")
	case Buy:
		fmt.Println("Broker Create Buy")
	case Sell:
		fmt.Println("Broker Create Sell")
	default:
		fmt.Println("Broker No indicator")
	}
	return nil
}
