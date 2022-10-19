package strategy

type Indicator struct {
	Direction int
}

const (
	None  = 0
	Close = 1
	Buy   = 2
	Sell  = 3
)
