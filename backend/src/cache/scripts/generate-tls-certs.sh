#!/bin/bash

# Script to generate self-signed TLS certificates for local cache-server development
# This creates certificates in the backend/src/cache/certs directory

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CERT_DIR="${SCRIPT_DIR}/../certs"

# Create certs directory if it doesn't exist
mkdir -p "${CERT_DIR}"

echo "Generating self-signed TLS certificates for cache-server..."
echo "Output directory: ${CERT_DIR}"

# Generate private key
openssl genrsa -out "${CERT_DIR}/key.pem" 2048

# Generate certificate signing request
openssl req -new -key "${CERT_DIR}/key.pem" -out "${CERT_DIR}/csr.pem" \
  -subj "/C=US/ST=CA/L=SF/O=Kubeflow/OU=Dev/CN=cache-server.kubeflow.svc"

# Generate self-signed certificate (valid for 365 days)
openssl x509 -req -days 365 -in "${CERT_DIR}/csr.pem" \
  -signkey "${CERT_DIR}/key.pem" -out "${CERT_DIR}/cert.pem"

# Clean up CSR
rm "${CERT_DIR}/csr.pem"

echo ""
echo "✅ TLS certificates generated successfully!"
echo "   Certificate: ${CERT_DIR}/cert.pem"
echo "   Private Key: ${CERT_DIR}/key.pem"
echo ""
echo "These certificates are for LOCAL DEVELOPMENT ONLY."
echo "Do not use in production!"
