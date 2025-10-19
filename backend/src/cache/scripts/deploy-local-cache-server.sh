#!/bin/bash

# Script to build and deploy local cache-server image to Kind cluster
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/../../../.."
NAMESPACE="${NAMESPACE:-kubeflow}"
IMAGE_NAME="cache-server"
IMAGE_TAG="local-dev"
FULL_IMAGE="${IMAGE_NAME}:${IMAGE_TAG}"
KIND_REGISTRY="kind-registry:5000"

echo "🔨 Building and Deploying Local Cache Server to Kind"
echo "===================================================="
echo

echo "1️⃣  Building cache-server Docker image..."
cd "$PROJECT_ROOT"
docker build -t "${FULL_IMAGE}" -f backend/Dockerfile.cacheserver .
echo "   ✅ Image built: ${FULL_IMAGE}"

echo
echo "2️⃣  Tagging image for Kind registry..."
docker tag "${FULL_IMAGE}" "${KIND_REGISTRY}/${FULL_IMAGE}"
echo "   ✅ Tagged as: ${KIND_REGISTRY}/${FULL_IMAGE}"

echo
echo "3️⃣  Pushing image to Kind registry..."
docker push "${KIND_REGISTRY}/${FULL_IMAGE}"
echo "   ✅ Image pushed to Kind registry"

echo
echo "4️⃣  Updating cache-server deployment..."
kubectl set image deployment/cache-server \
  -n "$NAMESPACE" \
  cache-server="${KIND_REGISTRY}/${FULL_IMAGE}"
echo "   ✅ Deployment image updated"

echo
echo "5️⃣  Waiting for rollout to complete..."
kubectl rollout status deployment/cache-server -n "$NAMESPACE" --timeout=180s
echo "   ✅ Rollout complete"

echo
echo "6️⃣  Checking Pod status..."
kubectl get pods -n "$NAMESPACE" -l app=cache-server

echo
echo "7️⃣  Checking logs (last 20 lines)..."
echo "   ─────────────────────────────────────────────"
kubectl logs -n "$NAMESPACE" -l app=cache-server --tail=20 || true
echo "   ─────────────────────────────────────────────"

echo
echo "8️⃣  Verifying database connection..."
if kubectl logs -n "$NAMESPACE" -l app=cache-server | grep -q "Database created"; then
    echo "   ✅ Cache server successfully connected to PostgreSQL"
else
    echo "   ❌ Database connection may have failed"
    exit 1
fi

if kubectl logs -n "$NAMESPACE" -l app=cache-server | grep -q "Cache server is ready"; then
    echo "   ✅ Cache server is ready to accept requests"
else
    echo "   ⚠️  'Cache server is ready' message not found (may be using old code)"
fi

echo
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ Local Cache Server Deployed Successfully!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo
echo "Deployed image: ${KIND_REGISTRY}/${FULL_IMAGE}"
echo "Namespace: $NAMESPACE"
