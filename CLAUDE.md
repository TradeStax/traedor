# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Traedor is a Go-based algorithmic trading system designed for backtesting and live trading, with a focus on futures markets. It uses a pluggable, interface-driven architecture that separates data feeds, trading strategies, and broker implementations.

## Key Commands

### Build and Run
```bash
# Build the project
go build -o bin/traedor

# Run (requires config.yaml)
./bin/traedor

# Build and run
./run.sh

# Development with hot reload (requires Air)
./run-air.sh
```

### Configuration
Before running, copy `example-config.yaml` to `config.yaml` and configure:
- Datafeed settings (Sierra Chart data paths)
- Broker parameters (starting balance, fees, margins)
- Strategy configuration (study data paths and indicators)

## Architecture

The system follows a pipeline architecture with these core components:

1. **Trader** (`pkg/trader/`) - Orchestrates all components
2. **Datafeed** (`pkg/datafeed/`) - Provides market data via channels
   - Supports: CSV, Sierra Chart (SC), TD Ameritrade, Generated data
3. **Strategy** (`pkg/strategy/`) - Analyzes data and generates trading signals
   - Pre-built: SMA, RSI, MACD, SC (Sierra Chart), Ensemble
   - Custom strategies: Modify `pkg/strategy/scStrategy.go`
4. **Broker** (`pkg/broker/`) - Executes trades based on signals
   - Futures broker with profit taking and stop loss strategies

### Key Interfaces
- `IAuthHelper` - Authentication for brokers/datafeeds
- `IBroker` - Trading and account management
- `IDatafeed` - Market data provider
- `IStrategy` - Trading signal generation

### Data Flow
```
Datafeed → (market data) → Trader → (price data) → Strategy → (signals) → Broker
```

## Important Notes

- **No test files exist** - The project currently lacks unit tests
- Configuration is managed via Viper library (`internal/config/`)
- Uses Go channels for asynchronous communication between components
- Designed for futures trading with proper margin and fee management
- Sierra Chart integration requires downloading tick data and study values separately