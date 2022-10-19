package broker

type IBroker interface {
	GetAccountStats() (Account, error)
	SendTrade(Trade) error
}
