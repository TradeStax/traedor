package datafeed

import (
	"fmt"
	"log"

	"github.com/tradestax/go-tdameritrade"
	"github.com/tradestax/traedor/auth"
	"github.com/tradestax/traedor/config"
)

type TDADatafeed struct {
	authHelper *auth.TDAAuthHelper
	config     config.Config
	dataChan   chan Data
	errorChan  chan error
}

func NewTDADatafeed(c config.Config, ah auth.IAuthHelper) *TDADatafeed {
	authHelper, ok := ah.(*auth.TDAAuthHelper)
	if !ok {
		panic(fmt.Errorf("Failed to convert IAuthHelper to TDAAuthHelper"))
	}
	df := &TDADatafeed{
		authHelper: authHelper,
		config:     c,
		dataChan:   make(chan Data),
		errorChan:  make(chan error),
	}
	return df
}

func (d *TDADatafeed) GetDatafeed() chan Data {
	go d.tdaDatafeed()
	if err := d.subscribe(); err != nil {
		panic(err)
	}
	return d.dataChan
}

func (d *TDADatafeed) GetErrorChan() chan error {
	return d.errorChan
}

// TODO:
// Parse response and determine error case
func (d *TDADatafeed) subscribe() error {
	d.authHelper.StreamingClient.SendCommand(tdameritrade.Command{
		Requests: []tdameritrade.StreamRequest{
			{
				Service:   "CHART_EQUITY",
				Requestid: "2",
				Command:   "SUBS",
				Account:   d.authHelper.UPN.Accounts[0].AccountID,
				Source:    d.authHelper.UPN.StreamerInfo.AppID,
				Parameters: tdameritrade.StreamParams{
					Keys:   "SPY,$SPX.X",
					Fields: "0,1,2,3,4,5,6",
				},
			},
		},
	})

	d.authHelper.StreamingClient.SendCommand(tdameritrade.Command{
		Requests: []tdameritrade.StreamRequest{
			{
				Service:   "QUOTE",
				Requestid: "3",
				Command:   "SUBS",
				Account:   d.authHelper.UPN.Accounts[0].AccountID,
				Source:    d.authHelper.UPN.StreamerInfo.AppID,
				Parameters: tdameritrade.StreamParams{
					Keys:   "SPY",
					Fields: "0,1,2,3,4,5,6,7,8",
				},
			},
		},
	})
	return nil
}

func (d *TDADatafeed) tdaDatafeed() {
	defer d.authHelper.StreamingClient.Close()
	messages, errors := d.authHelper.StreamingClient.ReceiveText()
	for {
		select {
		case message := <-messages:
			log.Printf("message: %s", message)

		case err := <-errors:
			log.Printf("error: %v", err)
			return
		}
	}
}
