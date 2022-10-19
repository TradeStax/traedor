package broker

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
	return nil
}
