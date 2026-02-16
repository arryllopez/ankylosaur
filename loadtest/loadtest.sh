#!/bin/bash

# Load testing script for ankyloGo
# This script runs load tests against the baseline and middleware endpoints
# to measure throughput and overhead

echo "==================================="
echo "ankyloGo Load Test"
echo "==================================="
echo ""

# Check if hey is installed
if ! command -v hey &> /dev/null
then
    echo "ERROR: 'hey' is not installed."
    echo ""
    echo "Install it with:"
    echo "  go install github.com/rakyll/hey@latest"
    echo ""
    exit 1
fi

echo "Starting load test server in background..."
go run loadtest/server.go &
SERVER_PID=$!
sleep 2  # Wait for server to start

echo ""
echo "==================================="
echo "Test 1: Baseline (no middleware)"
echo "==================================="
hey -n 100000 -c 100 -q 10000 http://localhost:8080/baseline

echo ""
echo "==================================="
echo "Test 2: With ankyloGo middleware"
echo "==================================="
hey -n 100000 -c 100 -q 10000 http://localhost:8080/with-middleware

echo ""
echo "Stopping server..."
kill $SERVER_PID

echo ""
echo "==================================="
echo "Load test complete!"
echo "==================================="
echo "Compare the 'Requests/sec' values to calculate overhead."
echo "Overhead = Baseline latency - Middleware latency"
