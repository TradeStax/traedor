#!/bin/bash

echo "Running Traedor Backend Tests"
echo "============================="

# Add testify dependency if not present
echo "Checking test dependencies..."
go mod tidy

# Add testify if needed
if ! grep -q "github.com/stretchr/testify" go.mod; then
    echo "Adding testify dependency..."
    go get github.com/stretchr/testify/assert
    go get github.com/stretchr/testify/require
fi

echo ""
echo "Running all Go tests..."

# Run tests with verbose output and coverage
go test -v -cover ./pkg/storage/
go test -v -cover ./pkg/broker/
go test -v -cover ./pkg/trader/
go test -v -cover ./pkg/signal/

echo ""
echo "Running tests for all packages with race detection..."
go test -race ./...

echo ""
echo "Test Summary Complete"