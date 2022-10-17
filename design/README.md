# Traedor High Level Design

## Traedor is designed to be a pluggable architecture where different implementations of each componenent can be swapped out to work in different environments. And example of this would be leveraging a paper trading broker and CSV datafeed for backtesting where a TDAmeritrade broker and datafeed module could be used to a live trading environment.

```mermaid
  graph TD;
    A[Trader];
    B[(DataFeed)] --Send Data--> A;
    C[Broker];
    D[Strategy];
    A --Send Trade--> C;
    A --Get Indicator--> D;
```

```mermaid
  classDiagram
    class Trader
      Trader : "NewTrader(Params) *Trader"
      Trader : Run(*chan bool)

    class DataFeed
      DataFeed : NewDataFeed(Params) *DataFeed
      DataFeed : GetDataFeed() *chan *Data

    class Broker
      Broker : NewBroker(Params) *Broker
      Broker : GetAccountStats() *Account
      Broker : SendTrade(Trade) Result

    class Strategy
      Strategy : NewStrategy(Params) *Strategy
      Strategy : AddData(*Data) error
      Strategy : GetIndicatorFeed() *chan Indicator
```
