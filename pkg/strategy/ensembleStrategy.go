package strategy

import (
	"fmt"

	"github.com/tradestax/traedor/pkg/datafeed"
	"github.com/tradestax/traedor/pkg/strategy/types"
)

type EnsembleStrategy struct {
	config        []types.Config
	indicatorChan chan types.Indicator
	indicators    []types.Indicator
	members       []types.IStrategy
	memberChans   []chan types.Indicator
	ignores       []bool
}

func NewEnsembleStrategy(c []types.Config, ic chan types.Indicator) *EnsembleStrategy {
	// number of members in ensemble
	nMembers := len(c)
	// create ensemble struct
	es := EnsembleStrategy{
		config:        c,
		indicatorChan: ic,
		indicators:    make([]types.Indicator, nMembers),
		members:       make([]types.IStrategy, nMembers),
		memberChans:   make([]chan types.Indicator, nMembers),
		ignores:       make([]bool, nMembers),
	}
	// create channels to receive indicators
	for i := range es.memberChans {
		es.memberChans[i] = make(chan types.Indicator, 1)
	}
	// create the member strategies
	for i := range es.members {
		es.members[i] = NewStrategy([]types.Config{c[i]}, es.memberChans[i])
		es.ignores[i] = c[i].IgnoreNone
	}
	return &es
}

func (s *EnsembleStrategy) AddData(data datafeed.Data) error {
	for _, m := range s.members {
		err := m.AddData(data)
		if err != nil {
			return fmt.Errorf("Failed sending data to ensemble member: %w", err)
		}
	}
	s.determineIndicator()
	return nil
}

func (s *EnsembleStrategy) GetIndicatorFeed() chan types.Indicator {
	return s.indicatorChan
}

func (s *EnsembleStrategy) determineIndicator() {
	var ind types.Indicator
	for i, mChan := range s.memberChans {
		s.indicators[i] = <-mChan
	}
	ind.Direction = s.indicators[0].Direction
	ind.Price = s.indicators[0].Price
	for i, mInd := range s.indicators {
		if mInd.Direction == types.None && s.ignores[i] {
			continue
		}
		if mInd.Direction != ind.Direction {
			ind.Direction = types.None
			break
		}
	}
	s.indicatorChan <- ind
}
