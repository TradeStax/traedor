package broker

type Trade struct {
	Symbol    string
	Operation int
	Price     float64
	Time      int64
}

const (
	None  = 0
	Close = 1
	Buy   = 2
	Sell  = 3
)
