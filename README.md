# Traedor

## Getting Started

See `example-config.yaml` for some quickstart options. This must be copied to `config.yaml` before running

## Running the backtester against Sierra Chart Data

* Download the data from Sierra Chart, I personally use Intraday Tick Data for this
  * Update the `Datafeed` section in `config.yaml` with the path for this data

* Download study values from Sierra Chart
  * This should be "bar data with study values"
  * Update `Strategy` section in `config.yaml` with the path for this data

## Building & Running

* [Install golang](https://go.dev/doc/install)
* `go build -o bin/traedor`
* `./bin/traedor`

## Creating a Strategy

* Update `pkg/strategy/scStrategy.go` `determineIndicator` function with your desired logic
* Follow "Building & Running" section to rebuild

## Upcoming

- [ ] Add GitHub Release to avoid needing to build to use
- [ ] Improved Documentation for using Sierra Chart Data
- [ ] Improved Documentation on creating Strategies
- [ ] Config drive strategy

## Source

* [auth](./pkg/auth/) module handles authenitcate to remote datafeeds and brokers
* [broker](./pkg/broker/) module handles trading and account management
* [config](./internal/config/) module handles configuration management
* [assets](./assets/) stores pre-canned test data
* [datafeed](./pkg/datafeed/) module provides the trader with data
* [strategy](./pkg/strategy/) module provides indicators based on data
* [trader](./pkg/trader/) modules brings all of the pieces together

