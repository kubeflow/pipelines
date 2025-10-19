#!/bin/bash
# Script to build and deploy cache-server to Kind cluster for integration tests
# Usage: ./deploy-to-kind.sh [cluster-name]

set -euo pipefail

CLUSTER_NAME="${1:-dev-pipelines-api}"
NAMESPACE="kubeflow"
IMAGE_NAME="kfp-cache-server-local"
IMAGE_TAG="test-$(date +%Y%m%d-%H%M%S)"

echo "🚀 Deploying cache-server to Kind cluster: ${CLUSTER_NAME}"
echo "📦 Building ARM64-compatible Docker image..."

# Change to repo root
cd "$(git rev-parse --show-toplevel)"

# Build cache-server binary for linux/arm64
echo "🔨 Building cache-server binary..."
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build \
  -o ./backend/src/cache/bin/cache_server \
  ./backend/src/cache/*.go

# Create a simple Dockerfile for the binary
cat > ./backend/src/cache/bin/Dockerfile <<'EOF'
FROM alpine:3.19
RUN adduser -S appuser
USER appuser
WORKDIR /bin
COPY cache_server /bin/cache_server
ENTRYPOINT ["/bin/cache_server"]
EOF

# Build Docker image
echo "🐳 Building Docker image..."
docker build -t ${IMAGE_NAME}:${IMAGE_TAG} \
  -f ./backend/src/cache/bin/Dockerfile \
  ./backend/src/cache/bin/

# Load image into Kind cluster
echo "📥 Loading image into Kind cluster..."
kind load docker-image ${IMAGE_NAME}:${IMAGE_TAG} --name ${CLUSTER_NAME}

# Update deployment to use new image
echo "🔄 Updating cache-server deployment..."
kubectl --context kind-${CLUSTER_NAME} set image \
  deployment/cache-server \
  server=${IMAGE_NAME}:${IMAGE_TAG} \
  -n ${NAMESPACE}

# Wait for rollout
echo "⏳ Waiting for deployment to be ready..."
kubectl --context kind-${CLUSTER_NAME} rollout status \
  deployment/cache-server \
  -n ${NAMESPACE} \
  --timeout=120s

# Check pod status
echo "✅ Cache-server deployed successfully!"
kubectl --context kind-${CLUSTER_NAME} get pods -n ${NAMESPACE} -l app=cache-server

# Show logs
echo ""
echo "📋 Recent logs:"
kubectl --context kind-${CLUSTER_NAME} logs -n ${NAMESPACE} \
  deployment/cache-server \
  --tail=20

echo ""
echo "✨ Deployment complete! Image: ${IMAGE_NAME}:${IMAGE_TAG}"
