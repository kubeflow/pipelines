#!/bin/bash

# Script to verify local cache-server setup
# This checks all prerequisites before running the cache-server

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERT_DIR="${SCRIPT_DIR}/../certs"

echo "🔍 Verifying Cache Server Local Setup..."
echo ""

# Check 1: Kubernetes cluster
echo "1️⃣  Checking Kubernetes cluster connection..."
if kubectl cluster-info &> /dev/null; then
    echo "   ✅ Kubernetes cluster is accessible"
else
    echo "   ❌ Cannot connect to Kubernetes cluster"
    echo "      Run: kind create cluster (if using kind)"
    exit 1
fi

# Check 2: PostgreSQL in kubeflow namespace
echo ""
echo "2️⃣  Checking PostgreSQL service in kubeflow namespace..."
if kubectl get svc postgres-service -n kubeflow &> /dev/null; then
    echo "   ✅ postgres-service found"
    kubectl get svc postgres-service -n kubeflow | tail -n 1
else
    echo "   ❌ postgres-service not found in kubeflow namespace"
    echo "      Deploy PostgreSQL first"
    exit 1
fi

# Check 3: PostgreSQL pod running
echo ""
echo "3️⃣  Checking PostgreSQL pod status..."
POD_STATUS=$(kubectl get pods -n kubeflow -l app=postgres -o jsonpath='{.items[0].status.phase}' 2>/dev/null || echo "NotFound")
if [ "$POD_STATUS" == "Running" ]; then
    echo "   ✅ PostgreSQL pod is running"
else
    echo "   ❌ PostgreSQL pod is not running (status: $POD_STATUS)"
    exit 1
fi

# Check 4: TLS certificates
echo ""
echo "4️⃣  Checking TLS certificates..."
if [ -f "${CERT_DIR}/cert.pem" ] && [ -f "${CERT_DIR}/key.pem" ]; then
    echo "   ✅ TLS certificates exist"
    echo "      cert: ${CERT_DIR}/cert.pem"
    echo "      key:  ${CERT_DIR}/key.pem"
else
    echo "   ⚠️  TLS certificates not found"
    echo "   Generating certificates now..."
    ${SCRIPT_DIR}/generate-tls-certs.sh
fi

# Check 5: Database credentials
echo ""
echo "5️⃣  Checking PostgreSQL credentials..."
if kubectl get secret postgres-secret -n kubeflow &> /dev/null; then
    DB_USER=$(kubectl get secret postgres-secret -n kubeflow -o jsonpath='{.data.username}' | base64 -d)
    echo "   ✅ postgres-secret found"
    echo "      Username: ${DB_USER}"
else
    echo "   ⚠️  postgres-secret not found, using defaults (user/password)"
fi

# Check 6: Port 5432 availability
echo ""
echo "6️⃣  Checking if port 5432 is available..."
if lsof -i :5432 &> /dev/null; then
    echo "   ⚠️  Port 5432 is already in use"
    echo "      This might be port-forward already running (which is fine)"
    lsof -i :5432
else
    echo "   ✅ Port 5432 is available"
fi

# Check 7: Port 8443 availability
echo ""
echo "7️⃣  Checking if port 8443 is available..."
if lsof -i :8443 &> /dev/null; then
    echo "   ❌ Port 8443 is already in use"
    echo "      Stop the process using this port:"
    lsof -i :8443
    exit 1
else
    echo "   ✅ Port 8443 is available"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "✅ All checks passed! Ready to run cache-server"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Next steps:"
echo "1. In VSCode, open Debug panel (Cmd+Shift+D)"
echo "2. Select 'PG-Launch Cache Server'"
echo "3. Press F5 to start debugging"
echo ""
echo "Or run manually:"
echo "  kubectl port-forward -n kubeflow svc/postgres-service 5432:5432 &"
echo "  cd backend/src/cache && go run . -db_driver=pgx -db_host=127.0.0.1 ..."
