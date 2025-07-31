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

# Get signal parameters
echo
echo -e "${BLUE}=== Signal Generator Configuration ===${NC}"
$DOCKER_CMD exec postgres psql -U traedor -d traedor -c "
WITH signal_configs AS (
    SELECT 
        jsonb_array_elements(config->'signals_with_params') as signal_config
    FROM runs 
    WHERE id = '$RUN_ID'
)
SELECT 
    signal_config->>'signal_definition_id' as signal_type,
    jsonb_pretty(signal_config->'parameters') as parameters
FROM signal_configs;
"

# Show signal parameters in a more readable format
echo
echo -e "${BLUE}=== Signal Parameters Summary ===${NC}"
SIGNAL_PARAMS=$($DOCKER_CMD exec postgres psql -U traedor -d traedor -t -A -c "
WITH signal_configs AS (
    SELECT 
        jsonb_array_elements(config->'signals_with_params') as signal_config
    FROM runs 
    WHERE id = '$RUN_ID'
)
SELECT 
    signal_config->>'signal_definition_id' as signal_type,
    signal_config->'parameters' as params
FROM signal_configs;
" 2>/dev/null)

if [ -n "$SIGNAL_PARAMS" ]; then
    while IFS='|' read -r signal_type params; do
        echo -e "${GREEN}Signal Type: ${YELLOW}$signal_type${NC}"
        
        case "$signal_type" in
            "sma_crossover")
                short_period=$(echo "$params" | jq -r '.short_period // "N/A"')
                long_period=$(echo "$params" | jq -r '.long_period // "N/A"')
                aggregation=$(echo "$params" | jq -r '.aggregation_interval // "N/A"')
                echo "  - Short Period: $short_period"
                echo "  - Long Period: $long_period"
                echo "  - Aggregation Interval: $aggregation"
                ;;
            "rsi")
                period=$(echo "$params" | jq -r '.period // "N/A"')
                overbought=$(echo "$params" | jq -r '.overbought_level // "N/A"')
                oversold=$(echo "$params" | jq -r '.oversold_level // "N/A"')
                echo "  - Period: $period"
                echo "  - Overbought Level: $overbought"
                echo "  - Oversold Level: $oversold"
                ;;
            *)
                echo "  - Parameters: $params"
                ;;
        esac
        echo
    done <<< "$SIGNAL_PARAMS"
fi

# Get signal count
echo
SIGNAL_COUNT=$($DOCKER_CMD exec postgres psql -U traedor -d traedor -t -c "SELECT COUNT(*) FROM signals WHERE run_id = '$RUN_ID';" 2>/dev/null | tr -d ' ' || echo "0")

echo -e "${BLUE}=== Signal Summary ===${NC}"
echo -e "Total signals generated: ${YELLOW}$SIGNAL_COUNT${NC}"

# Show signals data
if [ "$SIGNAL_COUNT" -gt 0 ]; then
    echo
    echo -e "${BLUE}=== Signals Generated During Backtest ===${NC}"
    $DOCKER_CMD exec postgres psql -U traedor -d traedor -c "
    SELECT 
        s.time,
        s.symbol,
        CASE 
            WHEN s.direction = 0 THEN 'None'
            WHEN s.direction = 1 THEN 'Close'
            WHEN s.direction = 2 THEN 'Buy'
            WHEN s.direction = 3 THEN 'Sell'
            ELSE 'Unknown'
        END as direction,
        s.price,
        sd.name as signal_name,
        sd.type as signal_type
    FROM signals s
    LEFT JOIN signal_definitions sd ON s.signal_definition_id = sd.id
    WHERE s.run_id = '$RUN_ID'
    ORDER BY s.time
    LIMIT 20;
    "

    if [ "$SIGNAL_COUNT" -gt 20 ]; then
        echo
        echo -e "${YELLOW}Showing first 20 signals out of $SIGNAL_COUNT total.${NC}"
    fi

    # Signal statistics
    echo
    echo -e "${BLUE}=== Signal Statistics ===${NC}"
    $DOCKER_CMD exec postgres psql -U traedor -d traedor -c "
    SELECT 
        CASE 
            WHEN direction = 0 THEN 'None'
            WHEN direction = 1 THEN 'Close'
            WHEN direction = 2 THEN 'Buy'
            WHEN direction = 3 THEN 'Sell'
            ELSE 'Unknown'
        END as signal_direction,
        COUNT(*) as count
    FROM signals 
    WHERE run_id = '$RUN_ID'
    GROUP BY direction
    ORDER BY direction;
    "

    # Signals by definition
    echo
    echo -e "${BLUE}=== Signals by Definition ===${NC}"
    $DOCKER_CMD exec postgres psql -U traedor -d traedor -c "
    SELECT 
        COALESCE(sd.name, 'Unknown') as signal_name,
        COALESCE(sd.type, 'Unknown') as signal_type,
        COUNT(s.*) as signal_count
    FROM signals s
    LEFT JOIN signal_definitions sd ON s.signal_definition_id = sd.id
    WHERE s.run_id = '$RUN_ID'
    GROUP BY sd.name, sd.type
    ORDER BY signal_count DESC;
    "
fi

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

# Show signal-to-trade correlation if both exist
if [ "$SIGNAL_COUNT" -gt 0 ] && [ "$TRADE_COUNT" -gt 0 ]; then
    echo
    echo -e "${BLUE}=== Signal to Trade Correlation ===${NC}"
    $DOCKER_CMD exec postgres psql -U traedor -d traedor -c "
    WITH signal_trades AS (
        SELECT 
            s.time as signal_time,
            s.direction as signal_direction,
            s.price as signal_price,
            t.open_time as trade_time,
            t.operation as trade_operation,
            t.open_price as trade_price,
            ABS(EXTRACT(EPOCH FROM (t.open_time - s.time))) as time_diff_seconds
        FROM signals s
        CROSS JOIN trades t
        WHERE s.run_id = '$RUN_ID' 
        AND t.run_id = '$RUN_ID'
        AND ABS(EXTRACT(EPOCH FROM (t.open_time - s.time))) <= 60  -- Within 60 seconds
        AND (
            (s.direction = 2 AND t.operation = 'BUY') OR
            (s.direction = 3 AND t.operation = 'SELL')
        )
    )
    SELECT 
        signal_time,
        CASE 
            WHEN signal_direction = 2 THEN 'Buy'
            WHEN signal_direction = 3 THEN 'Sell'
        END as signal,
        signal_price,
        trade_time,
        trade_operation,
        trade_price,
        ROUND(time_diff_seconds::numeric, 2) as seconds_to_trade
    FROM signal_trades
    ORDER BY signal_time
    LIMIT 10;
    "
    
    echo
    echo -e "${YELLOW}Note: Showing signals that were followed by trades within 60 seconds (first 10).${NC}"
fi

echo
echo -e "${GREEN}Done!${NC}"