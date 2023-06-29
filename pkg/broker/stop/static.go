package stop

type StaticConfig struct {
	Offset float64
}

type StaticStop struct {
	direction int
	fillPrice float64
	stopPrice float64
}

func NewStaticStop(c *Config) IStop {
	ss := &StaticStop{
		direction: c.Direction,
		fillPrice: c.FillPrice,
		stopPrice: float64(0.0),
	}
	ss.init(c.Static.Offset)
	return ss
}

func (s *StaticStop) Stop(price float64) bool {
	return s.isStop(price)
}

func (s *StaticStop) init(offset float64) {
	switch s.direction {
	case Buy:
		s.stopPrice = s.fillPrice - offset
	case Sell:
		s.stopPrice = s.fillPrice + offset
	}
}

func (s *StaticStop) isStop(price float64) bool {
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
