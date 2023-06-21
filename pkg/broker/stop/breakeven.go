package stop

type BreakevenStop struct {
	stopPrice    float64
	triggerPrice float64
	offset       float64
}

func NewBreakevenStop() IStop {
	return &BreakevenStop{}
}

func (s *BreakevenStop) Stop(price float64) bool {
	return false
}
