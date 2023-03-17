package strategy

type Indicator struct {
	Direction int
	Price     float64
	Time      int64
}

const (
	None  = 0
	Close = 1
	Buy   = 2
	Sell  = 3
)
