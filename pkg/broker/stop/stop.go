package stop

type stopBuilder func() IStop

type IStop interface {
	Stop(float64) bool
}

var stops = map[string]stopBuilder{
	"breakeven": NewBreakevenStop,
}

func NewStop(stop string) IStop {
	f := stops[stop]
	return f()
}
