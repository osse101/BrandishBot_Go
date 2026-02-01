#!/bin/bash

# scripts/unit_tests.sh
# Runs unit tests (skipping integration tests via -short flag)s

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

TARGET=$1

if [ -z "$TARGET" ]; then
    TARGET="./..."
    echo -e "${YELLOW}Running all unit tests...${NC}"
else
    echo -e "${YELLOW}Running unit tests for $TARGET...${NC}"
fi

# Run tests with -short flag to skip integration tests
# Use -v for verbose output only if failed, to keep it clean? 
# Or just standard output. Let's use standard output but maybe colorize if we could.
# For now, just run standard go test -short.

go test -short -cover "$TARGET"
EXIT_CODE=$?

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "\n${GREEN}All unit tests passed!${NC}"
else
    echo -e "\n${RED}Unit tests failed!${NC}"
fi

exit $EXIT_CODE
