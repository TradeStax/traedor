# Traedor

## Source

* [auth](./auth/) module handles authenitcate to remote datafeeds and brokers
* [broker](./broker/) module handles trading and account management
* [config](./config/) module handles configuration management
* [data](./data/) stores pre-canned test data
* [datafeed](./datafeed/) module provides the trader with data
* [reports](./reports/) module collects statistics about the trader and provides reports
* [strategy](./strategy/) module provides indicators based on data
* [trader](./trader/) modules brings all of the pieces together

See `example-config.yaml` for some quickstart options. This must be copied to `config.yaml` before running
