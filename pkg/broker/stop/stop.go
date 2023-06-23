package stop

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
	Breakeven BreakevenConfig
	Static    StaticConfig
}

type stopBuilder func(*Config) IStop

type IStop interface {
	Stop(float64) bool
}

var stops = map[string]stopBuilder{
	"breakeven": NewBreakevenStop,
	"static":    NewStaticStop,
}

func NewStop(c *Config) IStop {
	f, ok := stops[c.Type]
	if !ok {
		return nil
	}
	return f(c)
}
