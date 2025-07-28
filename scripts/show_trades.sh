#!/bin/bash

# Script to show raw trade data from the database for a specific backtest run ID
# Usage: ./show_trades.sh <run_id>

set -e

# Check if run_id is provided
if [ $# -eq 0 ]; then
    echo "Usage: $0 <run_id>"
    echo "Example: $0 5850323a-ebea-4a6b-81a8-a0b2bef5aaa3"
    exit 1
fi

RUN_ID="$1"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Raw Trade Data for Run ID: ${YELLOW}$RUN_ID${BLUE} ===${NC}"
echo

# Check if Docker Compose is available
if command -v docker >/dev/null 2>&1 && docker compose version >/dev/null 2>&1; then
    DOCKER_CMD="docker compose"
elif command -v docker-compose >/dev/null 2>&1; then
    DOCKER_CMD="docker-compose"
else
    echo -e "${RED}Error: Neither 'docker compose' nor 'docker-compose' found${NC}"
    exit 1
fi

# Check if postgres container is running
if ! $DOCKER_CMD ps | grep -q "traedor-postgres"; then
    echo -e "${RED}Error: PostgreSQL container is not running${NC}"
    echo "Please start it with: $DOCKER_CMD up -d postgres"
    exit 1
fi

# First, verify the run exists
echo -e "${BLUE}Checking if run exists...${NC}"
RUN_EXISTS=$($DOCKER_CMD exec postgres psql -U traedor -d traedor -t -c "SELECT COUNT(*) FROM runs WHERE id = '$RUN_ID';" 2>/dev/null | tr -d ' ' || echo "0")

if [ -z "$RUN_EXISTS" ] || [ "$RUN_EXISTS" -eq 0 ]; then
    echo -e "${RED}Error: Run ID '$RUN_ID' not found${NC}"
    echo
    echo -e "${YELLOW}Available runs:${NC}"
    $DOCKER_CMD exec postgres psql -U traedor -d traedor -c "SELECT id, status, created_at FROM runs ORDER BY created_at DESC LIMIT 10;"
    exit 1
fi

echo -e "${GREEN}Run found!${NC}"
echo

# Get run details
echo -e "${BLUE}=== Run Details ===${NC}"
$DOCKER_CMD exec postgres psql -U traedor -d traedor -c "
SELECT 
    id,
    status,
    progress,
    started_at,
    completed_at,
    created_at
FROM runs 
WHERE id = '$RUN_ID';
"

echo
echo -e "${BLUE}=== Run Configuration ===${NC}"
$DOCKER_CMD exec postgres psql -U traedor -d traedor -c "
SELECT 
    config->>'symbol' as symbol,
    config->>'timeframe' as timeframe,
    config->>'start_time' as start_time,
    config->>'end_time' as end_time,
    config->'broker'->>'starting_balance' as starting_balance
FROM runs 
WHERE id = '$RUN_ID';
"

# Get trade count
echo
TRADE_COUNT=$($DOCKER_CMD exec postgres psql -U traedor -d traedor -t -c "SELECT COUNT(*) FROM trades WHERE run_id = '$RUN_ID';" 2>/dev/null | tr -d ' ' || echo "0")

echo -e "${BLUE}=== Trade Summary ===${NC}"
echo -e "Total trades: ${YELLOW}$TRADE_COUNT${NC}"

if [ "$TRADE_COUNT" -eq 0 ]; then
    echo -e "${YELLOW}No trades found for this run.${NC}"
    exit 0
fi

# Show raw trade data
echo
echo -e "${BLUE}=== Raw Trade Data ===${NC}"
$DOCKER_CMD exec postgres psql -U traedor -d traedor -c "
SELECT 
    symbol,
    operation,
    quantity,
    open_price,
    close_price,
    open_time,
    close_time,
    net_profit,
    max_profit,
    max_drawdown,
    mfe,
    mae,
    mfe_percent,
    mae_percent
FROM trades 
WHERE run_id = '$RUN_ID'
ORDER BY open_time
LIMIT 20;
"

if [ "$TRADE_COUNT" -gt 20 ]; then
    echo
    echo -e "${YELLOW}Showing first 20 trades out of $TRADE_COUNT total.${NC}"
    echo -e "To see all trades, use: ${BLUE}$DOCKER_CMD exec postgres psql -U traedor -d traedor -c \"SELECT * FROM trades WHERE run_id = '$RUN_ID' ORDER BY open_time;\"${NC}"
fi

# Show trade statistics
echo
echo -e "${BLUE}=== Trade Statistics ===${NC}"
$DOCKER_CMD exec postgres psql -U traedor -d traedor -c "
SELECT 
    COUNT(*) as total_trades,
    COUNT(CASE WHEN net_profit > 0 THEN 1 END) as winning_trades,
    COUNT(CASE WHEN net_profit < 0 THEN 1 END) as losing_trades,
    ROUND(SUM(net_profit)::numeric, 2) as total_pnl,
    ROUND(AVG(net_profit)::numeric, 4) as avg_pnl,
    ROUND(MIN(net_profit)::numeric, 2) as worst_trade,
    ROUND(MAX(net_profit)::numeric, 2) as best_trade,
    ROUND(AVG(open_price)::numeric, 2) as avg_open_price,
    ROUND(AVG(close_price)::numeric, 2) as avg_close_price
FROM trades 
WHERE run_id = '$RUN_ID';
"

# Show price range analysis
echo
echo -e "${BLUE}=== Price Analysis ===${NC}"
$DOCKER_CMD exec postgres psql -U traedor -d traedor -c "
SELECT 
    ROUND(MIN(open_price)::numeric, 2) as min_open_price,
    ROUND(MAX(open_price)::numeric, 2) as max_open_price,
    ROUND(MIN(close_price)::numeric, 2) as min_close_price,
    ROUND(MAX(close_price)::numeric, 2) as max_close_price,
    ROUND((MAX(open_price) - MIN(open_price))::numeric, 2) as price_range
FROM trades 
WHERE run_id = '$RUN_ID' AND close_price IS NOT NULL;
"

echo
echo -e "${GREEN}Done!${NC}"