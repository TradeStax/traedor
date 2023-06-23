package stop

type BreakevenConfig struct {
	Trigger float64
	Offset  float64
}

type BreakevenStop struct {
	direction    int
	fillPrice    float64
	stopPrice    float64
	trigger      float64
	triggerPrice float64
	offset       float64
}

func NewBreakevenStop(c *Config) IStop {
	bs := &BreakevenStop{
		direction:    c.Direction,
		fillPrice:    c.FillPrice,
		stopPrice:    float64(0.0),
		trigger:      c.Breakeven.Trigger,
		triggerPrice: float64(0.0),
		offset:       c.Breakeven.Offset,
	}
	bs.init()
	return bs
}

func (s *BreakevenStop) Stop(price float64) bool {
	s.updateStop(price)
	return s.isStop(price)
}

func (s *BreakevenStop) init() {
	switch s.direction {
	case Buy:
		s.triggerPrice = s.fillPrice + s.trigger
	case Sell:
		s.triggerPrice = s.fillPrice - s.trigger
	}
}

func (s *BreakevenStop) updateStop(price float64) {
	switch s.direction {
	case Buy:
		if price >= s.triggerPrice {
			s.stopPrice = s.fillPrice + s.offset
		}
	case Sell:
		if price <= s.triggerPrice {
			s.stopPrice = s.fillPrice - s.offset
		}
	}
}

func (s *BreakevenStop) isStop(price float64) bool {
	if s.stopPrice == float64(0.0) {
		return false
	}
	switch s.direction {
	case Buy:
		if price <= s.stopPrice {
			return true
		}
	case Sell:
		if price >= s.stopPrice {
			return true
		}
	}
	return false
}
