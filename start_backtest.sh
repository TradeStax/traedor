#!/bin/bash

# Simple Traedor Backtest Starter - Returns just the URL
set -e

# Configuration  
API_BASE="http://10.10.199.96:8080/api"
FRONTEND_BASE="http://10.10.199.96:3030"

# Backtest configuration
BACKTEST_CONFIG='{
  "symbol": "/MES",
  "timeframe": "tick",
  "start_time": "2023-01-01T00:00:00Z", 
  "end_time": "2023-03-13T00:00:00Z",
  "datafeeds": [
    {
      "type": "Database",
      "symbol": "/MES",
      "data_path": "",
      "interval": "tick"
    }
  ],
  "broker": {
    "type": "Futures", 
    "starting_balance": 10000,
    "weekly_withdrawl": 0,
    "trailing_stop_amount": 10,
    "fee_per_side": 1,
    "open_slippage": 0.25,
    "symbol": {
      "name": "/MES",
      "margin": 500,
      "point_price": 5
    }
  },
  "strategies": [],
  "signals": [
    "c0114404-d3ce-4370-b336-b35afdd32e9a"
  ]
}'

# Create backtest and get run ID
RUN_ID=$(curl -s -X POST \
  -H "Content-Type: application/json" \
  -d "$BACKTEST_CONFIG" \
  "$API_BASE/runs" | jq -r '.id')

# Return the URL
echo "$FRONTEND_BASE/runs/$RUN_ID"