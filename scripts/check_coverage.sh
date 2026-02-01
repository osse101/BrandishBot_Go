#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C.UTF-8

# Usage: ./check_coverage.sh <coverage_file> <threshold>

COVERAGE_FILE="${1:-}"
THRESHOLD="${2:-}"

if [ -z "$COVERAGE_FILE" ]; then
    echo "Error: Coverage file path required."
    echo "Usage: $0 <coverage_file> <threshold>"
    exit 1
fi

if [ -z "$THRESHOLD" ]; then
    THRESHOLD=80
fi

if [ ! -f "$COVERAGE_FILE" ]; then
    echo "Error: Coverage file '$COVERAGE_FILE' not found."
    echo "Run tests first to generate it."
    exit 1
fi

echo "Checking coverage threshold ($THRESHOLD%)..."

# Calculate coverage
COVERAGE=$(go tool cover -func="$COVERAGE_FILE" | grep total | awk '{print $3}' | sed 's/%//')

if [ -z "$COVERAGE" ]; then
    echo "Error: Could not determine coverage."
    exit 1
fi

echo "Total Coverage: ${COVERAGE}%"

# Compare with threshold
# Use awk for floating point comparison
PASS=$(echo "$COVERAGE $THRESHOLD" | awk '{if ($1 < $2) print 0; else print 1}')

if [ "$PASS" -eq 1 ]; then
    echo "✅ Coverage meets threshold."
    exit 0
else
    echo "❌ Coverage is below threshold."
    exit 1
fi
