#!/bin/bash

# Manual Webhook Testing Script for Cache Server
# Prerequisites: Cache server running on https://localhost:8443

# To run this test
# cd backend/src/cache/scripts
#./manual-webhook-test.sh
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TEST_DATA_DIR="${SCRIPT_DIR}/../test-data"
CACHE_SERVER_URL="https://localhost:8443/mutate"
DB_HOST="127.0.0.1"
DB_USER="user"
DB_PASSWORD="password"
DB_NAME="cachedb"

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "🧪 Cache Server Manual Webhook Testing"
echo "========================================"
echo

# Check if cache server is running
echo "0️⃣  Checking if cache server is accessible..."
if curl -k -s -o /dev/null -w "%{http_code}" "$CACHE_SERVER_URL" | grep -q "405"; then
    echo -e "   ${GREEN}✅ Cache server is running${NC}"
else
    echo -e "   ${RED}❌ Cache server is not accessible at $CACHE_SERVER_URL${NC}"
    echo "   Please start cache server first (VSCode: PG-Launch Cache Server)"
    exit 1
fi

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 1: GET Request (Should be rejected)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
RESPONSE=$(curl -k -s "$CACHE_SERVER_URL")
if echo "$RESPONSE" | grep -q "Invalid method"; then
    echo -e "${GREEN}✅ PASS${NC}: Server correctly rejected GET request"
    echo "   Response: $RESPONSE"
else
    echo -e "${RED}❌ FAIL${NC}: Unexpected response"
    echo "   Response: $RESPONSE"
fi

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 2: Invalid Content-Type"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
RESPONSE=$(curl -k -s -X POST "$CACHE_SERVER_URL" \
  -H "Content-Type: text/plain" \
  -d "test")
if echo "$RESPONSE" | grep -q "Unsupported content type"; then
    echo -e "${GREEN}✅ PASS${NC}: Server correctly rejected invalid content type"
    echo "   Response: $RESPONSE"
else
    echo -e "${RED}❌ FAIL${NC}: Unexpected response"
    echo "   Response: $RESPONSE"
fi

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 3: Invalid JSON"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
RESPONSE=$(curl -k -s -X POST "$CACHE_SERVER_URL" \
  -H "Content-Type: application/json" \
  -d "invalid json")
if echo "$RESPONSE" | grep -q "Could not deserialize request"; then
    echo -e "${GREEN}✅ PASS${NC}: Server correctly rejected invalid JSON"
else
    echo -e "${RED}❌ FAIL${NC}: Unexpected response"
    echo "   Response: $RESPONSE"
fi

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 4: Valid AdmissionReview (Cache Miss)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Clear cache first
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" \
  -c "TRUNCATE TABLE execution_caches RESTART IDENTITY;" > /dev/null 2>&1

RESPONSE=$(curl -k -s -X POST "$CACHE_SERVER_URL" \
  -H "Content-Type: application/json" \
  -d @"${TEST_DATA_DIR}/test-admission-no-cache.json")

ALLOWED=$(echo "$RESPONSE" | jq -r '.response.allowed')
HAS_PATCH=$(echo "$RESPONSE" | jq -r '.response.patch' | grep -v null || echo "")

if [ "$ALLOWED" == "true" ] && [ -n "$HAS_PATCH" ]; then
    echo -e "${GREEN}✅ PASS${NC}: Request allowed with patch"

    # Decode and display patch
    echo
    echo -e "${BLUE}Decoded Patch:${NC}"
    echo "$RESPONSE" | jq -r '.response.patch' | base64 -d | jq .

    # Check for execution_cache_key
    CACHE_KEY=$(echo "$RESPONSE" | jq -r '.response.patch' | base64 -d | \
                jq -r '.[0].value."pipelines.kubeflow.org/execution_cache_key"' 2>/dev/null || echo "")

    if [ -n "$CACHE_KEY" ] && [ "$CACHE_KEY" != "null" ]; then
        echo -e "${GREEN}✅${NC} execution_cache_key added: ${CACHE_KEY:0:20}..."
    fi

    # Check cache_id is empty
    CACHE_ID=$(echo "$RESPONSE" | jq -r '.response.patch' | base64 -d | \
               jq -r '.[] | select(.path=="/metadata/labels") | .value."pipelines.kubeflow.org/cache_id"' 2>/dev/null || echo "")

    if [ "$CACHE_ID" == "" ] || [ "$CACHE_ID" == "null" ]; then
        echo -e "${GREEN}✅${NC} cache_id is empty (cache miss)"
    else
        echo -e "${YELLOW}⚠️${NC}  cache_id should be empty but got: $CACHE_ID"
    fi
else
    echo -e "${RED}❌ FAIL${NC}: Request not allowed or no patch returned"
    echo "   Response: $RESPONSE"
fi

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 5: Cache Hit (After inserting cache entry)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Insert cache entry
echo "Inserting cache entry into database..."
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" <<EOF > /dev/null 2>&1
INSERT INTO execution_caches
  ("ExecutionCacheKey", "ExecutionTemplate", "ExecutionOutput", "MaxCacheStaleness", "StartedAtInSec", "EndedAtInSec")
VALUES (
  '1933d178a14bc415466cfd1b3ca2100af975e8c59e1ff9d502fcf18eb5cbd7f7',
  '{"container":{"command":["echo","Hello"],"image":"python:3.9"}}',
  '{"workflows.argoproj.io/outputs":"{\"exitCode\":\"0\"}","pipelines.kubeflow.org/metadata_execution_id":"123"}',
  -1,
  EXTRACT(EPOCH FROM NOW())::bigint,
  EXTRACT(EPOCH FROM NOW())::bigint
);
EOF

# Verify insertion
CACHE_COUNT=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" \
  -t -c "SELECT COUNT(*) FROM execution_caches;")
echo "Cache entries in DB: $CACHE_COUNT"

# Send request again
RESPONSE=$(curl -k -s -X POST "$CACHE_SERVER_URL" \
  -H "Content-Type: application/json" \
  -d @"${TEST_DATA_DIR}/test-admission-no-cache.json")

# Check for cache hit indicators
PATCH_DECODED=$(echo "$RESPONSE" | jq -r '.response.patch' | base64 -d)

# Check if container was replaced
CONTAINER_IMAGE=$(echo "$PATCH_DECODED" | jq -r '.[] | select(.path=="/spec/containers") | .value[0].image' 2>/dev/null || echo "")

if echo "$CONTAINER_IMAGE" | grep -q "busybox"; then
    echo -e "${GREEN}✅ PASS${NC}: Container replaced with busybox (cache hit)"
    echo "   Replacement image: $CONTAINER_IMAGE"
else
    echo -e "${RED}❌ FAIL${NC}: Container not replaced"
    echo "   Got image: $CONTAINER_IMAGE"
fi

# Check cache_id is set
CACHE_ID=$(echo "$PATCH_DECODED" | \
           jq -r '.[] | select(.path=="/metadata/labels") | .value."pipelines.kubeflow.org/cache_id"' 2>/dev/null || echo "")

if [ -n "$CACHE_ID" ] && [ "$CACHE_ID" != "" ] && [ "$CACHE_ID" != "null" ]; then
    echo -e "${GREEN}✅${NC} cache_id set: $CACHE_ID"
else
    echo -e "${RED}❌${NC} cache_id not set"
fi

# Check reused_from_cache label
REUSED=$(echo "$PATCH_DECODED" | \
         jq -r '.[] | select(.path=="/metadata/labels") | .value."pipelines.kubeflow.org/reused_from_cache"' 2>/dev/null || echo "")

if [ "$REUSED" == "true" ]; then
    echo -e "${GREEN}✅${NC} reused_from_cache label set"
else
    echo -e "${YELLOW}⚠️${NC}  reused_from_cache label not found"
fi

echo
echo -e "${BLUE}Full Patch:${NC}"
echo "$PATCH_DECODED" | jq .

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 6: TFX Pod (Should be skipped)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

RESPONSE=$(curl -k -s -X POST "$CACHE_SERVER_URL" \
  -H "Content-Type: application/json" \
  -d @"${TEST_DATA_DIR}/test-admission-tfx.json")

HAS_PATCH=$(echo "$RESPONSE" | jq -r '.response.patch' 2>/dev/null || echo "null")

if [ "$HAS_PATCH" == "null" ] || [ -z "$HAS_PATCH" ]; then
    echo -e "${GREEN}✅ PASS${NC}: TFX pod skipped (no patch returned)"
else
    echo -e "${YELLOW}⚠️  UNEXPECTED${NC}: Patch returned for TFX pod"
    echo "   Patch: $(echo $HAS_PATCH | base64 -d 2>/dev/null || echo $HAS_PATCH)"
fi

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST 7: KFP V2 Pod (Should be skipped)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

RESPONSE=$(curl -k -s -X POST "$CACHE_SERVER_URL" \
  -H "Content-Type: application/json" \
  -d @"${TEST_DATA_DIR}/test-admission-v2.json")

HAS_PATCH=$(echo "$RESPONSE" | jq -r '.response.patch' 2>/dev/null || echo "null")

if [ "$HAS_PATCH" == "null" ] || [ -z "$HAS_PATCH" ]; then
    echo -e "${GREEN}✅ PASS${NC}: V2 pod skipped (no patch returned)"
else
    echo -e "${YELLOW}⚠️  UNEXPECTED${NC}: Patch returned for V2 pod"
    echo "   Patch: $(echo $HAS_PATCH | base64 -d 2>/dev/null || echo $HAS_PATCH)"
fi

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📊 Database Verification"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

echo "Current cache entries:"
PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -U "$DB_USER" -d "$DB_NAME" \
  -c "SELECT \"ID\", LEFT(\"ExecutionCacheKey\", 20) || '...' as CacheKey, \"MaxCacheStaleness\" FROM execution_caches;"

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Manual Testing Complete!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo
echo "To clean up test data, run:"
echo "  PGPASSWORD=password psql -h 127.0.0.1 -U user -d cachedb \\"
echo "    -c \"TRUNCATE TABLE execution_caches RESTART IDENTITY;\""
