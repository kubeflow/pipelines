#!/bin/bash

# Script to verify cache-server deployment in Kind cluster
set -e

NAMESPACE="${NAMESPACE:-kubeflow}"

echo "🔍 Verifying Cache Server Deployment in Kind Cluster"
echo "================================================"
echo

# Check if cache-server deployment exists
echo "1️⃣  Checking cache-server deployment..."
if kubectl get deployment cache-server -n "$NAMESPACE" &> /dev/null; then
    echo "   ✅ cache-server deployment exists"
    kubectl get deployment cache-server -n "$NAMESPACE"
else
    echo "   ❌ cache-server deployment not found"
    exit 1
fi

echo
echo "2️⃣  Checking cache-server pod status..."
POD_STATUS=$(kubectl get pods -n "$NAMESPACE" -l app=cache-server -o jsonpath='{.items[0].status.phase}' 2>/dev/null || echo "NotFound")
if [ "$POD_STATUS" == "Running" ]; then
    echo "   ✅ cache-server pod is Running"
    kubectl get pods -n "$NAMESPACE" -l app=cache-server
else
    echo "   ❌ cache-server pod is not running (status: $POD_STATUS)"
    kubectl get pods -n "$NAMESPACE" -l app=cache-server || true
    exit 1
fi

echo
echo "3️⃣  Checking cache-server image..."
CACHE_IMAGE=$(kubectl get deployment cache-server -n "$NAMESPACE" -o jsonpath='{.spec.template.spec.containers[0].image}')
echo "   Image: $CACHE_IMAGE"

echo
echo "4️⃣  Checking cache-server logs (last 20 lines)..."
echo "   ─────────────────────────────────────────────"
kubectl logs -n "$NAMESPACE" -l app=cache-server --tail=20 || true
echo "   ─────────────────────────────────────────────"

echo
echo "5️⃣  Checking cache-server service..."
if kubectl get svc cache-server -n "$NAMESPACE" &> /dev/null; then
    echo "   ✅ cache-server service exists"
    kubectl get svc cache-server -n "$NAMESPACE"
else
    echo "   ⚠️  cache-server service not found (this might be okay if using different naming)"
fi

echo
echo "6️⃣  Checking MutatingWebhookConfiguration..."
WEBHOOK_NAME="cache-webhook-kubeflow"
if kubectl get mutatingwebhookconfiguration "$WEBHOOK_NAME" &> /dev/null; then
    echo "   ✅ MutatingWebhookConfiguration exists"
    kubectl get mutatingwebhookconfiguration "$WEBHOOK_NAME" -o jsonpath='{.webhooks[0].clientConfig.service}' | jq .
else
    echo "   ⚠️  MutatingWebhookConfiguration not found"
    echo "      This is needed for cache to work!"
fi

echo
echo "7️⃣  Checking database connection from logs..."
if kubectl logs -n "$NAMESPACE" -l app=cache-server | grep -q "Database created"; then
    echo "   ✅ Cache server successfully connected to PostgreSQL"
else
    echo "   ❌ 'Database created' not found in logs"
    echo "      Cache server may have failed to connect to database"
fi

if kubectl logs -n "$NAMESPACE" -l app=cache-server | grep -q "Table: execution_caches"; then
    echo "   ✅ execution_caches table created/verified"
else
    echo "   ⚠️  'Table: execution_caches' not found in logs"
fi

echo
echo "8️⃣  Testing database connection directly..."
kubectl port-forward -n "$NAMESPACE" svc/postgres-service 5432:5432 --address=127.0.0.1 > /dev/null 2>&1 &
PF_PID=$!
sleep 2

if PGPASSWORD=password psql -h 127.0.0.1 -U user -d cachedb -c "SELECT COUNT(*) FROM execution_caches;" &> /dev/null; then
    CACHE_COUNT=$(PGPASSWORD=password psql -h 127.0.0.1 -U user -d cachedb -t -c "SELECT COUNT(*) FROM execution_caches;")
    echo "   ✅ Database accessible, cache entries: $CACHE_COUNT"
else
    echo "   ❌ Unable to connect to database"
fi

kill $PF_PID 2>/dev/null || true

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Cache Server Deployment Verification Complete"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
