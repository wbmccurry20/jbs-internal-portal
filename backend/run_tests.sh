#!/bin/bash

# JBS Internal Portal - Test Runner
# Runs all backend tests with coverage

set -e

echo "🧪 Running JBS Internal Portal Tests"
echo "=================================="

cd "$(dirname "$0")/.."

# Run tests with coverage
echo ""
echo "Running unit tests..."
go test ./internal/... -v -cover -coverprofile=coverage.out

# Display coverage summary
echo ""
echo "📊 Coverage Summary:"
go tool cover -func=coverage.out | grep total

# Optional: Generate HTML coverage report
if [ "$1" = "--html" ]; then
    echo ""
    echo "Generating HTML coverage report..."
    go tool cover -html=coverage.out -o coverage.html
    echo "✅ Coverage report saved to coverage.html"
fi

echo ""
echo "✅ All tests passed!"
