package datafeed

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"

	"github.com/tradestax/go-tdameritrade"
	"github.com/tradestax/traedor/pkg/auth"
)

type TDADatafeed struct {
	authHelper *auth.TDAAuthHelper
	config     *Config
	dataChan   chan Data
	errorChan  chan error
}

type TdaData struct {
	Data     []TdaDataMsg     `json:"data"`
	Snapshot []TdaSnapshotMsg `json:"snapshot"`
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
	F6      float64 `json:"6"`
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

type TdaSnapshotMsg struct {
	Content []TdaSnapshotContent `json:"content"`
	Service string               `json:"service"`
}

type TdaSnapshotContent struct {
	Data []TdaSnapshotDataMsg `json:"3"`
}

type TdaSnapshotDataMsg struct {
	F0 int64   `json:"0"`
	F1 float64 `json:"1"`
	F2 float64 `json:"2"`
	F3 float64 `json:"3"`
	F4 float64 `json:"4"`
	F5 float64 `json:"5"`
}

type historicalData []TdaSnapshotDataMsg

func (d historicalData) Len() int {
	return len(d)
}

func (d historicalData) Less(i, j int) bool {
	return d[i].F0 < d[j].F0
}

func (d historicalData) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func NewTDADatafeed(c *Config, ah auth.IAuthHelper, dc chan Data, ec chan error) *TDADatafeed {
	authHelper, ok := ah.(*auth.TDAAuthHelper)
	if !ok {
		log.Fatalln("Failed to convert IAuthHelper to TDAAuthHelper")
	}
	df := &TDADatafeed{
		authHelper: authHelper,
		config:     c,
		dataChan:   dc,
		errorChan:  ec,
	}
	return df
}

func (d *TDADatafeed) GetDatafeed() chan Data {
	return d.dataChan
}

func (d *TDADatafeed) Start() {
	go d.tdaDatafeed()
	if err := d.subscribe(); err != nil {
		log.Fatal(err)
	}
}

func (d *TDADatafeed) GetErrorChan() chan error {
	return d.errorChan
}

// TODO:
// Parse response and determine error case
func (d *TDADatafeed) subscribe() error {
	log.Printf("Subscribing to %v for %v\n", d.config.Service, d.config.Symbol)
	switch d.config.Service {
	case "CHART_HISTORY_FUTURES":
		d.authHelper.StreamingClient.SendCommand(tdameritrade.Command{
			Requests: []tdameritrade.StreamRequest{
				{
					Service:   d.config.Service,
					Requestid: "2",
					Command:   "GET",
					Account:   d.authHelper.UPN.Accounts[0].AccountID,
					Source:    d.authHelper.UPN.StreamerInfo.AppID,
					Parameters: tdameritrade.StreamParams{
						Symbol: d.config.Symbol,
						//StartTime: d.config.StartTime,
						//EndTime:   d.config.EndTime,
						Frequency: d.config.Interval,
					},
				},
			},
		})
	default:
		d.authHelper.StreamingClient.SendCommand(tdameritrade.Command{
			Requests: []tdameritrade.StreamRequest{
				{
					Service:   d.config.Service,
					Requestid: "2",
					Command:   "SUBS",
					Account:   d.authHelper.UPN.Accounts[0].AccountID,
					Source:    d.authHelper.UPN.StreamerInfo.AppID,
					Parameters: tdameritrade.StreamParams{
						Keys:   d.config.Symbol,
						Fields: d.config.Fields,
					},
				},
			},
		})
	}
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
						log.Println("QUOTE RECEIVED")
						continue
					case "CHART_EQUITY":
						// send to data channel
						for _, contentMsg := range dataMsg.Content {
							// for now hardcoded to only send SPY
							if contentMsg.Key == "SPY" {
								newData := Data{
									High:   contentMsg.F2,
									Low:    contentMsg.F3,
									Open:   contentMsg.F1,
									Close:  contentMsg.F4,
									Volume: contentMsg.F5,
									Symbol: contentMsg.Key,
								}
								if d.config.Print {
									newData.Print()
								}
								d.dataChan <- newData
							}
						}
					case "CHART_FUTURES":
						// send to data channel
						for _, contentMsg := range dataMsg.Content {
							newData := Data{
								High:   contentMsg.F3,
								Low:    contentMsg.F4,
								Open:   contentMsg.F2,
								Close:  contentMsg.F5,
								Volume: contentMsg.F6,
								Symbol: contentMsg.Key,
							}
							if d.config.Print {
								newData.Print()
							}
							d.dataChan <- newData
						}
					default:
						log.Printf("Data message received from unknown service %v\n", dataMsg.Service)
					}
				}
			} else if len(response.Snapshot) > 0 {
				for _, dataMsg := range response.Snapshot {
					switch dataMsg.Service {
					case "CHART_HISTORY_FUTURES":
						// send to data channel
						log.Printf("Received %v candles\n", len(dataMsg.Content[0].Data))
						var histData historicalData
						histData = dataMsg.Content[0].Data
						sort.Sort(histData)
						for _, contentMsg := range histData {
							newData := Data{
								Date:   contentMsg.F0,
								High:   contentMsg.F2,
								Low:    contentMsg.F3,
								Open:   contentMsg.F1,
								Close:  contentMsg.F4,
								Volume: contentMsg.F5,
								Symbol: d.config.Symbol,
							}
							if d.config.Print {
								newData.Print()
							}
							d.dataChan <- newData
						}
						d.errorChan <- fmt.Errorf("Test Completed")
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
