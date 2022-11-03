# Strategies

## Pre-Canned Strategies

There are a few pre-built strategies, this includes "SMA", "RSI", and "MACD". Please use these with caution as they are not intended to be high performing strategies, instead they are intended as building blocks for your own strategy.

## Ensemble Strategies

The ensemble strategy is used whenever more than one strategy is specified within the config. This multiplexes data to all of the strategies and returns a "BUY" or "SELL" signal when all strategies agree on the indicator. In all other cases "CLOSE" is returned.

## Custom Strategies

All files that end with `_custom.go` will be ignored by git. This allows you to develop custom strategies that will not be shared with others. To include these in the constructor map, add the following to one of your custom files.

``` go
func init() {
  customStrategies["custom"] = NewCustomStrategy
}
```

You can add as many different strategies as you would like.
