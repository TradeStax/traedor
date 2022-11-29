# Traedor

## Source

* [auth](./pkg/auth/) module handles authenitcate to remote datafeeds and brokers
* [broker](./pkg/broker/) module handles trading and account management
* [config](./internal/config/) module handles configuration management
* [assets](./assets/) stores pre-canned test data
* [datafeed](./pkg/datafeed/) module provides the trader with data
* [strategy](./pkg/strategy/) module provides indicators based on data
* [trader](./pkg/trader/) modules brings all of the pieces together

See `example-config.yaml` for some quickstart options. This must be copied to `config.yaml` before running
