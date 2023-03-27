package broker

import "log"

type Account struct {
	balance          float64
	availableBalance float64
	weeklyWithdrawl  float64
}

func (a *Account) Balance() float64 {
	return a.balance
}

func (a *Account) WeeklyWithdrawl() {
	log.Printf("Performing weekly withdrawl of %.2f\n", a.weeklyWithdrawl)
	a.balance = a.balance - a.weeklyWithdrawl
	a.availableBalance = a.availableBalance - a.weeklyWithdrawl
}
