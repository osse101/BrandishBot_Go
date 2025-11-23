#!/bin/bash

# scripts/run_tests.sh

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}Starting test session...${NC}"

# Ensure cleanup happens on exit
cleanup() {
    echo -e "\n${GREEN}Tearing down test environment...${NC}"
    # Stop and remove the specific test container if we created one, 
    # or use docker-compose down if we used that.
    # For this script, let's assume we want to run tests against the docker-compose environment
    # but we want to make sure we leave it clean.
    
    # If we want to be strict about "teardown AFTER a test session", we should probably
    # spin up a dedicated test environment.
    
    # However, the user request was "teardown the docker build after a test session".
    # This might mean "docker-compose down" after running tests.
    
    docker-compose down
}
trap cleanup EXIT

# Start environment
echo "Starting environment..."
docker-compose up -d db

# Wait for DB
echo "Waiting for database..."
sleep 5

# Apply migrations (just in case, or use the setup tool)
# We can run the setup tool locally if Go is installed, or via docker.
# Let's assume local Go for running tests as per previous steps.
if command -v go &> /dev/null; then
    echo "Applying migrations..."
    go run cmd/setup/main.go
else
    echo "${RED}Go not found. Cannot run tests locally.${NC}"
    exit 1
fi

# Run tests
echo "Running tests..."
go test ./...
TEST_EXIT_CODE=$?

if [ $TEST_EXIT_CODE -eq 0 ]; then
    echo -e "${GREEN}Tests passed!${NC}"
else
    echo -e "${RED}Tests failed!${NC}"
fi

exit $TEST_EXIT_CODE
