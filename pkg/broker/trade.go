package broker

import (
	"github.com/tradestax/traedor/pkg/types"
)

// Re-export constants from types package
const (
	None  = types.None
	Close = types.Close
	Buy   = types.Buy
	Sell  = types.Sell
)

// Trade is now defined in pkg/types
type Trade = types.Trade

// Re-export functions
var (
	Header             = types.Header
	WriteTradesToFile = types.WriteTradesToFile
)
