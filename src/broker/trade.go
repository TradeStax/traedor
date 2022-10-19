package broker

type Trade struct {
	Symbol    string
	Operation int
}

const (
	None  = 0
	Close = 1
	Buy   = 2
	Sell  = 3
)
