#!/bin/bash

# Script to verify that backtests are using the new streaming method
# Usage: ./verify_streaming.sh

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=== Verifying Streaming Implementation ===${NC}"
echo

# Check 1: Look for streaming initialization messages
echo -e "${BLUE}1. Checking for streaming initialization messages:${NC}"
streaming_init=$(docker compose logs backend --since 30m | grep -c "Starting true streaming" || echo "0")
if [ "$streaming_init" -gt 0 ]; then
    echo -e "  ✅ ${GREEN}Found $streaming_init 'Starting true streaming' messages${NC}"
else
    echo -e "  ❌ ${RED}No 'Starting true streaming' messages found${NC}"
fi

# Check 2: Look for individual tick streaming progress
echo -e "${BLUE}2. Checking for streaming progress messages:${NC}"  
streaming_progress=$(docker compose logs backend --since 30m | grep -c "Streamed.*ticks" || echo "0")
if [ "$streaming_progress" -gt 0 ]; then
    echo -e "  ✅ ${GREEN}Found $streaming_progress streaming progress messages${NC}"
    echo -e "  Latest progress:"
    docker compose logs backend | grep "Streamed.*ticks" | tail -3 | sed 's/^/    /'
else
    echo -e "  ❌ ${RED}No streaming progress messages found${NC}"
fi

# Check 3: Look for old chunked method (should NOT be present)
echo -e "${BLUE}3. Checking for old chunked method (should be absent):${NC}"
chunked_count=$(docker compose logs backend --since 30m 2>/dev/null | grep "Processing chunk" 2>/dev/null | wc -l || echo "0")
chunked_messages=$((chunked_count + 0))
if [ "$chunked_messages" -eq 0 ]; then
    echo -e "  ✅ ${GREEN}No 'Processing chunk' messages found (good!)${NC}"
else
    echo -e "  ❌ ${RED}Found $chunked_messages 'Processing chunk' messages - still using old method!${NC}"
    echo -e "  Recent chunk messages:"
    docker compose logs backend | grep "Processing chunk" | tail -3 | sed 's/^/    /'
fi

# Check 4: Current tick processing rate
echo -e "${BLUE}4. Current streaming performance:${NC}"
latest_tick=$(docker compose logs backend --tail=10 | grep "TimeAggregator: Tick" | tail -1 | grep -o "Tick #[0-9]*" | grep -o "[0-9]*" || echo "0")
if [ "$latest_tick" -gt 0 ]; then
    echo -e "  📊 ${YELLOW}Latest tick processed: #${latest_tick}${NC}"
    echo -e "  🚀 ${GREEN}Individual tick processing confirmed${NC}"
else
    echo -e "  ⚠️  ${YELLOW}No recent tick processing found${NC}"
fi

echo
echo -e "${BLUE}=== Summary ===${NC}"
if [ "$streaming_init" -gt 0 ] && [ "$streaming_progress" -gt 0 ] && [ "$chunked_messages" -eq 0 ]; then
    echo -e "🎉 ${GREEN}SUCCESS: Your backtest is using the new streaming method!${NC}"
    echo -e "   • True streaming initialization: ✅"
    echo -e "   • Individual tick streaming: ✅" 
    echo -e "   • No old chunked method: ✅"
else
    echo -e "❌ ${RED}Your backtest may still be using the old method${NC}"
    echo -e "   • Try rebuilding: docker compose build --no-cache backend"
    echo -e "   • Then restart: docker compose restart backend"
fi