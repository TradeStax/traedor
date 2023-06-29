package profit

const (
	None  = 0
	Close = 1
	Buy   = 2
	Sell  = 3
)

type Config struct {
	Type      string
	Direction int
	FillPrice float64
	Static    StaticConfig
}

type profitBuilder func(*Config) IProfit

type IProfit interface {
	Profit(float64) bool
}

var profits = map[string]profitBuilder{
	"static": NewStaticProfit,
}

func NewProfit(c *Config) IProfit {
	f, ok := profits[c.Type]
	if !ok {
		return nil
	}
	return f(c)
}
