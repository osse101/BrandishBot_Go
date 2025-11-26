#!/bin/bash
# scripts/check_deps.sh
# Checks for required dependencies

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo "Checking dependencies..."

# Check Go
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    echo -e "${GREEN}✅ Go installed: $GO_VERSION${NC}"
else
    echo -e "${RED}❌ Go not found!${NC}"
    echo "   Install from: https://go.dev/dl/"
    exit 1
fi

# Check Docker
if command -v docker &> /dev/null; then
    DOCKER_VERSION=$(docker --version | awk '{print $3}' | tr -d ',')
    echo -e "${GREEN}✅ Docker installed: $DOCKER_VERSION${NC}"
else
    echo -e "${RED}❌ Docker not found!${NC}"
    echo "   Install from: https://docs.docker.com/get-docker/"
    exit 1
fi

# Check Docker Compose
if command -v docker-compose &> /dev/null; then
    COMPOSE_VERSION=$(docker-compose --version | awk '{print $3}' | tr -d ',')
    echo -e "${GREEN}✅ Docker Compose installed: $COMPOSE_VERSION${NC}"
else
    echo -e "${YELLOW}⚠️  Docker Compose not found (optional if using 'docker compose')${NC}"
fi

# Check Make
if command -v make &> /dev/null; then
    MAKE_VERSION=$(make --version | head -n 1 | awk '{print $3}')
    echo -e "${GREEN}✅ Make installed: $MAKE_VERSION${NC}"
else
    echo -e "${RED}❌ Make not found!${NC}"
    echo "   Install via package manager (e.g., sudo apt install make)"
    exit 1
fi

# Check Goose (Optional)
if command -v goose &> /dev/null; then
    GOOSE_VERSION=$(goose --version | awk '{print $3}')
    echo -e "${GREEN}✅ Goose installed: $GOOSE_VERSION${NC}"
elif [ -f "$HOME/go/bin/goose" ]; then
    GOOSE_VERSION=$($HOME/go/bin/goose --version | awk '{print $3}')
    echo -e "${GREEN}✅ Goose installed (in ~/go/bin): $GOOSE_VERSION${NC}"
else
    echo -e "${YELLOW}⚠️  Goose not found (Recommended for dev)${NC}"
    echo "   Install: go install github.com/pressly/goose/v3/cmd/goose@v3.11.0"
fi

echo ""
echo -e "${GREEN}🎉 Environment check complete!${NC}"
