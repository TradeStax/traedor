package datafeed

import (
	"encoding/json"
	"log"

	"github.com/tradestax/go-tdameritrade"
	"github.com/tradestax/traedor/auth"
	"github.com/tradestax/traedor/config"
)

type TDADatafeed struct {
	authHelper *auth.TDAAuthHelper
	config     *config.Config
	dataChan   chan Data
	errorChan  chan error
}

type TdaData struct {
	Data     []TdaDataMsg     `json:"data"`
	Notify   []TdaNotifyMsg   `json:"notify"`
	Response []TdaResponseMsg `json:"response"`
}

type TdaDataMsg struct {
	Command   string              `json:"command"`
	Content   []TdaDataContentMsg `json:"content"`
	Service   string              `json:"service"`
	Timestamp int                 `json:"timestamp"`
}

type TdaDataContentMsg struct {
	Delayed bool    `json:"delayed"`
	Key     string  `json:"key"`
	Seq     int     `json:"seq"`
	F1      float64 `json:"1"`
	F2      float64 `json:"2"`
	F3      float64 `json:"3"`
	F4      float64 `json:"4"`
	F5      float64 `json:"5"`
	F6      int     `json:"6"`
	F7      int     `json:"7"`
	F8      int     `json:"8"`
}

type TdaNotifyMsg struct {
	Heartbeat string `json:"heartbeat"`
}

type TdaResponseMsg struct {
	Command   string                `json:"command"`
	Content   TdaResponseContentMsg `json:"content"`
	RequestId string                `json:"requestid"`
	Service   string                `json:"service"`
	Timestamp int                   `json:"timestamp"`
}

type TdaResponseContentMsg struct {
	Code    int    `json:"code"`
	Message string `json:"msg"`
}

func NewTDADatafeed(c *config.Config, ah auth.IAuthHelper) *TDADatafeed {
	authHelper, ok := ah.(*auth.TDAAuthHelper)
	if !ok {
		log.Fatalln("Failed to convert IAuthHelper to TDAAuthHelper")
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
		log.Fatal(err)
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
					Fields: "0,1,2,3,4,5,8",
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
			var response TdaData
			if err := json.Unmarshal(message, &response); err != nil {
				log.Println(err.Error())
			}
			// response message, validate success
			if len(response.Response) > 0 {
				if response.Response[0].Content.Code != 0 {
					log.Fatalf("Response code %v does not equal 0. Message: %v\n", response.Response[0].Content.Code, response.Response[0].Content.Message)
				}
			} else if len(response.Data) > 0 {
				// data message, format to Data and send to trader
				for _, dataMsg := range response.Data {
					switch dataMsg.Service {
					case "QUOTE":
						// ignore for now
						continue
					case "CHART_EQUITY":
						// send to data channel
						for _, contentMsg := range dataMsg.Content {
							// for now hardcoded to only send SPY
							if contentMsg.Key == "SPY" {
								d.dataChan <- Data{
									High:   contentMsg.F2,
									Low:    contentMsg.F3,
									Open:   contentMsg.F1,
									Close:  contentMsg.F4,
									Volume: contentMsg.F5,
									Symbol: contentMsg.Key,
								}
							}
						}
					default:
						log.Printf("Data message received from unknown service %v\n", dataMsg.Service)
					}
				}
			} else {
				continue
			}

		case err := <-errors:
			log.Printf("error: %v", err)
			return
		}
	}
}
