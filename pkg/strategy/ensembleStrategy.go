package strategy

import (
	"fmt"

	"github.com/tradestax/traedor/pkg/datafeed"
)

type EnsembleStrategy struct {
	config        []Config
	indicatorChan chan Indicator
	indicators    []Indicator
	members       []IStrategy
	memberChans   []chan Indicator
}

func NewEnsembleStrategy(c []Config, ic chan Indicator) *EnsembleStrategy {
	// number of members in ensemble
	nMembers := len(c)
	// create ensemble struct
	es := EnsembleStrategy{
		config:        c,
		indicatorChan: ic,
		indicators:    make([]Indicator, nMembers),
		members:       make([]IStrategy, nMembers),
		memberChans:   make([]chan Indicator, nMembers),
	}
	// create channels to receive indicators
	for i := range es.memberChans {
		es.memberChans[i] = make(chan Indicator, 1)
	}
	// create the member strategies
	for i := range es.members {
		es.members[i] = NewStrategy([]Config{c[i]}, es.memberChans[i])
	}
	return &es
}

func (s *EnsembleStrategy) AddData(data datafeed.Data) error {
	for _, m := range s.members {
		err := m.AddData(data)
		if err != nil {
			return fmt.Errorf("Failed sending data to ensemble member")
		}
	}
	s.determineIndicator()
	return nil
}

func (s *EnsembleStrategy) GetIndicatorFeed() chan Indicator {
	return s.indicatorChan
}

func (s *EnsembleStrategy) determineIndicator() {
	var ind Indicator
	for i, mChan := range s.memberChans {
		s.indicators[i] = <-mChan
	}
	ind.Direction = s.indicators[0].Direction
	ind.Price = s.indicators[0].Price
	for _, mInd := range s.indicators {
		if mInd.Direction != ind.Direction {
			ind.Direction = Close
			break
		}
	}
	s.indicatorChan <- ind
}
