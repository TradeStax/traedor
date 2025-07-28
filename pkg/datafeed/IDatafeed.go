package datafeed

type IDatafeed interface {
	GetDatafeed() chan Data
	GetErrorChan() chan error
	Start()
	Stop() error
}
