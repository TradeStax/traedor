package broker

type Account struct {
	balance          float64
	availableBalance float64
}

func (a Account) Balance() float64 {
	return a.balance
}
