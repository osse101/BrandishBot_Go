#!/usr/bin/env bash
# Test security features

set -euo pipefail

# Load API key from .env file
if [ ! -f .env ]; then
    echo "Error: .env file not found. Please create it with API_KEY set."
    exit 1
fi

# Extract API_KEY from .env
API_KEY=$(grep '^API_KEY=' .env | cut -d '=' -f2)

if [ -z "$API_KEY" ]; then
    echo "Error: API_KEY not found in .env file"
    exit 1
fi

BASE_URL="http://localhost:8080"

echo "=== Security Feature Tests ==="
echo

echo "Test 1: Request without API key (should fail with 401)"
curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/test" \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","platform":"twitch","platform_id":"12345"}'
echo " - No API key"
echo

echo "Test 2: Request with wrong API key (should fail with 401)"
curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/test" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: wrong_key" \
  -d '{"username":"testuser","platform":"twitch","platform_id":"12345"}'
echo " - Wrong API key"
echo

echo "Test 3: Request with valid API key (should succeed with 200/201)"
curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/test" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{"username":"testuser","platform":"twitch","platform_id":"12345"}'
echo " - Valid API key"
echo

echo "Test 4: Invalid platform (should fail with 400)"
curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/test" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{"username":"testuser","platform":"invalid_platform","platform_id":"12345"}'
echo " - Invalid platform"
echo

echo "Test 5: Username too long (should fail with 400)"
LONG_USERNAME=$(printf 'A%.0s' {1..200})
curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/test" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d '{"username":"'"$LONG_USERNAME"'","platform":"twitch","platform_id":"12345"}'
echo " - Username too long"
echo

echo "Test 6: Username with control characters (should fail with 400)"
curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/test" \
  -H "Content-Type: application/json" \
  -H "X-API-Key: $API_KEY" \
  -d $'{"username":"test\\nuser","platform":"twitch","platform_id":"12345"}'
echo " - Username with control chars"
echo

echo "Test 7: Valid platforms (should all succeed)"
for platform in twitch youtube discord; do
  CODE=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/test" \
    -H "Content-Type: application/json" \
    -H "X-API-Key: $API_KEY" \
    -d '{"username":"user_'$platform'","platform":"'$platform'","platform_id":"12345"}')
  echo "  - $platform: $CODE"
done
echo

echo "=== Security Tests Complete ==="
