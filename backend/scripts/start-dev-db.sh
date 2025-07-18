#!/usr/bin/env bash

set -euo pipefail

# Use DB_TYPE environment variable to choose the database (default: postgres)
DB_TYPE="${DB_TYPE:-postgres}"
# Allow overriding Helm release name; default is dev-<DB_TYPE>-db
RELEASE_NAME="${RELEASE_NAME:-dev-${DB_TYPE}-db}"
# Use DB_NAMESPACE to choose the namespace; default is <DB_TYPE>-test
DB_NAMESPACE="${DB_NAMESPACE:-${DB_TYPE}-test}"
# For example: DB_NAMESPACE=kubeflow DB_TYPE=postgres make start-dev-db

if [ "$DB_TYPE" = "postgres" ]; then
  # Use Bitnami PostgreSQL chart for CI-friendly deployment
  helm upgrade --install ${RELEASE_NAME} bitnami/postgresql \
    --namespace ${DB_NAMESPACE} --create-namespace \
    -f ./helm-charts/postgresql/values.yaml
else
  # Example: deploy MySQL using the Bitnami chart
  helm upgrade --install ${RELEASE_NAME} bitnami/mysql \
    --namespace ${DB_NAMESPACE} --create-namespace
fi

# Waiting for the service to be ready
kubectl -n ${DB_NAMESPACE} wait --for=condition=Ready pod -l app.kubernetes.io/instance=${RELEASE_NAME} --timeout=2m
echo "✅ ${DB_TYPE} is ready in namespace ${DB_NAMESPACE}"
