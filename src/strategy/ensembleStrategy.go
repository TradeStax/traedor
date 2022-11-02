package strategy

import (
	"fmt"

	"github.com/tradestax/traedor/config"
	"github.com/tradestax/traedor/datafeed"
)

type EnsembleStrategy struct {
	config        []config.StrategyConfig
	indicatorChan chan Indicator
	indicators    []Indicator
	members       []IStrategy
	memberChans   []chan Indicator
}

func NewEnsembleStrategy(c []config.StrategyConfig, ic chan Indicator) *EnsembleStrategy {
	// number of members in ensemble
	nMembers := len(c)
	// create ensemble struct
	es := EnsembleStrategy{
		config:        c,
		indicatorChan: ic,
		indicators:    make([]Indicator, nMembers),
		members:       make([]chan Indicator, nMembers),
		memberChans:   make([]IStrategy, nMembers),
	}
	// create channels to receive indicators
	for i := range es.memberChans {
		es.memberChans[i] = make(chan Indicator, 1)
	}
	// create the member strategies
	for i := range es.members {
		es.members[i] = NewStrategy([]config.StrategyConfig{c[i]}, es.memberChans[i])
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
	return nil
}

func (s *EnsembleStrategy) GetIndicatorFeed() chan Indicator {
	return s.indicatorChan
}

// TODO:
// This is challenging and clunky
// should probably use a single channel instead
func (s *EnsembleStrategy) determineIndicator() {
	for {
		select {
		case err := <-t.errorChan:
		}
	}
}
