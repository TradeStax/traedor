package profit

type StaticConfig struct {
	Offset float64
}

type StaticProfit struct {
	direction   int
	fillPrice   float64
	profitPrice float64
}

func NewStaticProfit(c *Config) IProfit {
	ss := &StaticProfit{
		direction:   c.Direction,
		fillPrice:   c.FillPrice,
		profitPrice: float64(0.0),
	}
	ss.init(c.Static.Offset)
	return ss
}

func (s *StaticProfit) Profit(price float64) bool {
	return s.isProfit(price)
}

func (s *StaticProfit) init(offset float64) {
	switch s.direction {
	case Buy:
		s.profitPrice = s.fillPrice + offset
	case Sell:
		s.profitPrice = s.fillPrice - offset
	}
}

func (s *StaticProfit) isProfit(price float64) bool {
	if s.profitPrice == float64(0.0) {
		return false
	}
	switch s.direction {
	case Buy:
		if price >= s.profitPrice {
			return true
		}
	case Sell:
		if price <= s.profitPrice {
			return true
		}
	}
	return false
}
