# Traedor Database Scripts

This directory contains utility scripts for querying and analyzing data in the Traedor database.

## test_streaming.sh

Tests and compares the streaming vs paginated approach for trade data retrieval.

### Usage

```bash
./scripts/test_streaming.sh <run_id>
```

### Example

```bash
./scripts/test_streaming.sh 5850323a-ebea-4a6b-81a8-a0b2bef5aaa3
```

### What it tests

1. **Paginated Approach**: Tests the original `/api/runs/{id}/trades` endpoint with pagination
2. **Streaming Approach**: Tests the new `/api/runs/{id}/trades/stream` endpoint that returns all trades
3. **Performance Comparison**: Measures response times for both approaches
4. **Data Verification**: Ensures both endpoints return identical data
5. **Network Efficiency**: Calculates how many requests would be needed for complete data

### Benefits of Streaming

- **No Pagination Limits**: Gets ALL trades in a single request
- **Better Performance**: Often faster than paginated requests
- **Reduced Network Overhead**: No multiple round trips needed
- **Memory Efficient**: Uses PostgreSQL's streaming capabilities with `lib/pq`

## show_trades.sh

Shows raw trade data from the database for a specific backtest run ID.

### Usage

```bash
./scripts/show_trades.sh <run_id>
```

### Example

```bash
./scripts/show_trades.sh 5850323a-ebea-4a6b-81a8-a0b2bef5aaa3
```

### What it shows

1. **Run Details**: Status, progress, timing information
2. **Run Configuration**: Symbol, timeframe, date range, starting balance
3. **Trade Summary**: Total number of trades
4. **Raw Trade Data**: First 20 trades with all fields:
   - Symbol, operation (2=Buy, 3=Sell), quantity
   - Open/close prices (in actual trading format, not database storage format)
   - Open/close times
   - P&L metrics (net_profit, max_profit, max_drawdown)
   - MFE/MAE values and percentages
5. **Trade Statistics**: Win/loss counts, total P&L, averages
6. **Price Analysis**: Price ranges and statistics

### Prerequisites

- Docker and Docker Compose must be installed
- PostgreSQL container must be running (`docker compose up -d postgres`)

### Notes

- The script automatically detects whether to use `docker compose` or `docker-compose`
- Shows first 20 trades by default (provides command to see all trades if more exist)
- Prices are displayed in actual trading format (e.g., 3856.75 for ES futures)
- Includes colored output for better readability
- Provides helpful error messages and suggestions

### Troubleshooting

If you get "Run ID not found", the script will show available runs to choose from.

If you get "PostgreSQL container is not running", start it with:
```bash
docker compose up -d postgres
```