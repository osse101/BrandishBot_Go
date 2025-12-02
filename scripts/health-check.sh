#!/bin/bash
set -e

# BrandishBot Health Check Script
# Usage: ./scripts/health-check.sh <environment>
# Example: ./scripts/health-check.sh staging

ENVIRONMENT="${1:-production}"
TIMEOUT=5
PORT=8080

if [[ "$ENVIRONMENT" == "staging" ]]; then
    PORT=8081
fi

# Test /healthz endpoint
if ! curl -sf --max-time "$TIMEOUT" "http://localhost:$PORT/healthz" > /dev/null 2>&1; then
    echo "Health check failed: /healthz endpoint not responding"
    exit 1
fi

# Check if response time is acceptable (< 1 second)
RESPONSE_TIME=$(curl -sf --max-time "$TIMEOUT" -w "%{time_total}" -o /dev/null "http://localhost:$PORT/healthz" 2>/dev/null || echo "999")
if (( $(echo "$RESPONSE_TIME > 1.0" | bc -l) )); then
    echo "Health check warning: slow response time (${RESPONSE_TIME}s)"
fi

echo "Health check passed"
exit 0
