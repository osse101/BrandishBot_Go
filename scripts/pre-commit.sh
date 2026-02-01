#!/usr/bin/env bash
set -euo pipefail
export LC_ALL=C.UTF-8

# BrandishBot - Advanced Pre-commit Git Hook
# Automatically runs checks and formatting before committing

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Ensure go bin is in path
export PATH=$PATH:$(go env GOPATH)/bin

echo -e "${YELLOW}🚀 Running pre-commit checks...${NC}"

# 1. Secret Scanning
echo -e "🔍 Checking for secrets..."
STAGED_FILES=$(git diff --cached --name-only --diff-filter=ACM)
# Search for Discord tokens, generic passwords, secret keys
# Filtering out false positives by looking for actual assignments
if echo "$STAGED_FILES" | xargs grep -E "((mfa\.[a-z0-9_-]{20,})|([a-z0-9_-]{24}\.[a-z0-9_-]{6}\.[a-z0-9_-]{27}))|(\b(password|secret|api_key|token|client_id|client_secret)\b\s*[:=]\s*['\"][^'\"]+['\"])" /dev/null 2>/dev/null; then
    echo -e "${RED}❌ Potential secrets found in staged files! Please review and remove.${NC}"
    exit 1
fi

# 2. Go Format
STAGED_GO_FILES=$(echo "$STAGED_FILES" | grep ".go$" || true)
if [ -n "$STAGED_GO_FILES" ]; then
    echo -e "🎨 Running go fmt..."
    for file in $STAGED_GO_FILES; do
        if [ -f "$file" ]; then
            go fmt "$file"
            git add "$file"
        fi
    done
fi

# 3. Generate Check (sqlc, mocks, go mod tidy)
# Check if we should run generation (if SQL, interfaces, go.mod, or progression config changed)
RUN_GEN=0
if echo "$STAGED_FILES" | grep -E "(\.sql$|interfaces\.go$|go\.mod$|go\.sum$|progression_tree\.json$)" > /dev/null; then
    RUN_GEN=1
fi

if [ $RUN_GEN -eq 1 ]; then
    echo -e "⚙️ Running 'make generate' (sqlc, mocks, tidy)..."
    # We run it and then check for uncommitted changes in tracked files
    make generate > /dev/null
    
    # Check if anything changed that wasn't staged (including go.mod updates from tidy)
    if ! git diff --exit-code > /dev/null; then
        echo -e "${RED}❌ 'make generate' produced changes that are not staged for commit.${NC}"
        echo -e "${YELLOW}Please stage the updated mocks, sqlc code, or go.mod/go.sum and try again.${NC}"
        # We don't auto-add these because mocks/sqlc changes should be reviewed
        exit 1
    fi
fi

# 4. Linting (Only on new changes)
echo -e "🧹 Running linter on changes..."
if command -v golangci-lint > /dev/null; then
    golangci-lint run --new-from-rev=HEAD ./... || {
        echo -e "${RED}❌ Linter failed. Please fix issues before committing.${NC}"
        exit 1
    }
else
    echo -e "${YELLOW}⚠️ golangci-lint not found in PATH or GOPATH/bin, skipping lint.${NC}"
fi

# 5. Fast Unit Tests
echo -e "🧪 Running unit tests..."
make unit || {
    echo -e "${RED}❌ Unit tests failed.${NC}"
    exit 1
}

echo -e "${GREEN}✅ All pre-commit checks passed!${NC}"
exit 0
