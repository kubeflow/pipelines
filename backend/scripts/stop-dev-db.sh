#!/usr/bin/env bash
set -euo pipefail

DB_TYPE="${DB_TYPE:-postgres}"
RELEASE_NAME="dev-${DB_TYPE}-db"
NAMESPACE="${DB_TYPE}-test"

helm uninstall "${RELEASE_NAME}" --namespace "${NAMESPACE}" || true
kubectl delete namespace "${NAMESPACE}" --ignore-not-found

echo "✅ Stopped and cleaned up ${DB_TYPE} in namespace ${NAMESPACE}"
