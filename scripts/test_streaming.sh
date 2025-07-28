#!/bin/bash

# Script to test the streaming vs paginated approach for trade data retrieval
# Usage: ./test_streaming.sh <run_id>

set -e

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

echo -e "${BLUE}=== Testing Streaming vs Paginated Trade Retrieval ===${NC}"
echo -e "Run ID: ${YELLOW}$RUN_ID${NC}"
echo

# Test 1: Paginated approach (original)
echo -e "${BLUE}1. Testing Paginated Approach (limit=100, offset=0)${NC}"
echo -e "Endpoint: ${GREEN}/api/runs/$RUN_ID/trades?limit=100&offset=0${NC}"

start_time=$(date +%s%N)
paginated_response=$(curl -s "http://localhost:8080/api/runs/$RUN_ID/trades?limit=100&offset=0")
end_time=$(date +%s%N)
paginated_duration=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds

paginated_trades_count=$(echo "$paginated_response" | jq -r '.trades | length')
paginated_total=$(echo "$paginated_response" | jq -r '.pagination.total')

echo -e "Response time: ${YELLOW}${paginated_duration}ms${NC}"
echo -e "Trades returned: ${YELLOW}${paginated_trades_count}${NC}"
echo -e "Total available: ${YELLOW}${paginated_total}${NC}"
echo -e "Memory usage: ${YELLOW}Limited by page size (100 trades max)${NC}"
echo

# Test 2: Streaming approach (new)
echo -e "${BLUE}2. Testing Streaming Approach (all trades)${NC}"
echo -e "Endpoint: ${GREEN}/api/runs/$RUN_ID/trades/stream${NC}"

start_time=$(date +%s%N)
streaming_response=$(curl -s "http://localhost:8080/api/runs/$RUN_ID/trades/stream")
end_time=$(date +%s%N)
streaming_duration=$(( (end_time - start_time) / 1000000 )) # Convert to milliseconds

streaming_trades_count=$(echo "$streaming_response" | jq -r '.trades | length')
streaming_total=$(echo "$streaming_response" | jq -r '.total')

echo -e "Response time: ${YELLOW}${streaming_duration}ms${NC}"
echo -e "Trades returned: ${YELLOW}${streaming_trades_count}${NC}"
echo -e "Total available: ${YELLOW}${streaming_total}${NC}"
echo -e "Memory usage: ${YELLOW}All trades loaded (no pagination limits)${NC}"
echo

# Comparison
echo -e "${BLUE}=== Comparison Results ===${NC}"
echo -e "Data completeness:"
if [ "$streaming_trades_count" -eq "$paginated_total" ]; then
    echo -e "  ✅ Streaming got ALL trades: ${GREEN}$streaming_trades_count${NC}"
    echo -e "  ⚠️  Paginated got limited trades: ${YELLOW}$paginated_trades_count${NC} (would need ${YELLOW}$(( (paginated_total + 99) / 100 ))${NC} requests for all)"
else
    echo -e "  ❌ Data mismatch! Streaming: $streaming_trades_count, Expected: $paginated_total"
fi

echo
echo -e "Performance:"
if [ "$streaming_duration" -lt "$paginated_duration" ]; then
    echo -e "  🚀 Streaming was faster: ${GREEN}${streaming_duration}ms${NC} vs ${YELLOW}${paginated_duration}ms${NC}"
elif [ "$streaming_duration" -eq "$paginated_duration" ]; then
    echo -e "  ⚖️  Same performance: ${YELLOW}${streaming_duration}ms${NC}"
else
    echo -e "  📊 Streaming took longer: ${YELLOW}${streaming_duration}ms${NC} vs ${GREEN}${paginated_duration}ms${NC}"
    echo -e "     (Expected for large datasets, but gets ALL data in one request)"
fi

echo
echo -e "Network efficiency:"
requests_needed=$(( (paginated_total + 99) / 100 ))
echo -e "  📡 Paginated approach needs: ${YELLOW}${requests_needed}${NC} requests for all data"
echo -e "  📡 Streaming approach needs: ${GREEN}1${NC} request for all data"
echo -e "  💾 Network savings: ${GREEN}$(( requests_needed - 1 ))${NC} fewer round trips"

echo
echo -e "${BLUE}=== Sample Data Verification ===${NC}"
first_paginated_price=$(echo "$paginated_response" | jq -r '.trades[0].OpenPrice')
first_streaming_price=$(echo "$streaming_response" | jq -r '.trades[0].OpenPrice')

echo -e "First trade open price:"
echo -e "  Paginated: ${YELLOW}\$${first_paginated_price}${NC}"
echo -e "  Streaming:  ${YELLOW}\$${first_streaming_price}${NC}"

if [ "$first_paginated_price" = "$first_streaming_price" ]; then
    echo -e "  ✅ ${GREEN}Prices match perfectly${NC}"
else
    echo -e "  ❌ ${RED}Price mismatch detected${NC}"
fi

echo
echo -e "${GREEN}=== Test Complete ===${NC}"