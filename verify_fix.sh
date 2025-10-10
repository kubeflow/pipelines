#!/bin/bash

echo "=== Verifying API Fix ==="
echo ""

echo "1. Testing ListRuns API..."
response=$(curl -s "http://localhost:8888/apis/v2beta1/runs?page_size=2")
echo "Response: $response"
echo ""

if echo "$response" | jq -e '.runs' > /dev/null 2>&1; then
    echo "✅ SUCCESS: 'runs' field is present in response"
    runs_count=$(echo "$response" | jq '.runs | length')
    echo "   Runs array length: $runs_count"
else
    echo "❌ FAILED: 'runs' field is missing from response"
fi

echo ""
echo "2. Testing integration test..."
cd /Users/yunkaili/Documents/personal/open_source/kubeflow_pipeline/kubeflow-pipelines/backend/test/v2/integration
go test -v -run TestCache/TestCacheRecurringRun -runIntegrationTests -cacheEnabled=true -timeout=10m 2>&1 | tail -30
